/*

rtop - the remote system monitoring utility

Copyright (c) 2015 RapidLoop

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
)

const VERSION = "1.0"
const DEFAULT_REFRESH = 5 // default refresh interval in seconds

var currentUser *user.User

//----------------------------------------------------------------------------
// Command-line processing

func usage(code int) {
	fmt.Printf(
		`rtop %s - (c) 2015 RapidLoop - MIT Licensed - http://rtop-monitor.org
rtop monitors server statistics over an ssh connection

Usage: rtop [-i private-key-file] [-l log-level] [-L log-file] [user@]host[:port] [interval]

	-i private-key-file
		Encoded private key file to use (default: ~/.ssh/id_*  if present)
	-l log-level
		Set logging level (DEBUG, INFO, WARN, ERROR, FATAL) (default: FATAL)
	-L log-file
		File to write logs to (default: stderr only)
	[user@]host[:port]
		the SSH server to connect to, with optional username and port
	interval
		refresh interval in seconds (default: %d)

`, VERSION, DEFAULT_REFRESH)
	os.Exit(code)
}

func shift(q []string) (ok bool, val string, qnew []string) {
	if len(q) > 0 {
		ok = true
		val = q[0]
		qnew = q[1:]
	}
	return
}

func parseCmdLine() (host string, port int, user, key string, interval time.Duration, logLevel, logFile string) {
	ok, arg, args := shift(os.Args)
	var argKey, argHost, argInt, argLogLevel, argLogFile string
	for ok {
		ok, arg, args = shift(args)
		if !ok {
			break
		}
		if arg == "-h" || arg == "--help" || arg == "--version" {
			usage(0)
		}
		if arg == "-i" {
			ok, argKey, args = shift(args)
			if !ok {
				usage(1)
			}
		} else if arg == "-l" {
			ok, argLogLevel, args = shift(args)
			if !ok {
				usage(1)
			}
		} else if arg == "-L" {
			ok, argLogFile, args = shift(args)
			if !ok {
				usage(1)
			}
		} else if len(argHost) == 0 {
			argHost = arg
		} else if len(argInt) == 0 {
			argInt = arg
		} else {
			usage(1)
		}
	}
	if len(argHost) == 0 || argHost[0] == '-' {
		usage(1)
	}

	// Set default log level
	if len(argLogLevel) == 0 {
		logLevel = "FATAL"
	} else {
		logLevel = argLogLevel
	}

	// Set log file
	logFile = argLogFile

	// key
	if len(argKey) != 0 {
		key = argKey
	} // else key remains ""

	// user, addr
	var addr string
	if i := strings.Index(argHost, "@"); i != -1 {
		user = argHost[:i]
		if i+1 >= len(argHost) {
			usage(1)
		}
		addr = argHost[i+1:]
	} else {
		// user remains ""
		addr = argHost
	}

	// addr -> host, port
	if p := strings.Split(addr, ":"); len(p) == 2 {
		host = p[0]
		var err error
		if port, err = strconv.Atoi(p[1]); err != nil {
			Fatal("bad port: %v", err)
			usage(1)
		}
		if port <= 0 || port >= 65536 {
			Fatal("bad port: %d", port)
			usage(1)
		}
	} else {
		host = addr
		// port remains 0
	}

	// interval
	if len(argInt) > 0 {
		i, err := strconv.ParseUint(argInt, 10, 64)
		if err != nil {
			Fatal("bad interval: %v", err)
			usage(1)
		}
		if i <= 0 {
			Fatal("bad interval: %d", i)
			usage(1)
		}
		interval = time.Duration(i) * time.Second
	} // else interval remains 0

	return
}

// validateOS - rtop only support for Linux system
func validateOS(client *ssh.Client) {
	Debug("Validating remote OS type")
	ostype, err := runCommand(client, "uname")
	if err != nil {
		Fatal("Failed to get OS type: %v", err)
		os.Exit(1)
	}
	//remove newline character
	ostype = strings.Trim(ostype, "\n")

	Info("Remote OS detected: %s", ostype)
	if !strings.EqualFold(ostype, "Linux") {
		Fatal("rtop not supported for %s system", ostype)
		os.Exit(1)
	}
	Debug("OS validation successful")
}

//----------------------------------------------------------------------------

func main() {
	// get params from command line
	host, port, username, key, interval, logLevel, logFile := parseCmdLine()

	// Initialize logging
	InitLogging(logLevel, true, logFile)
	Info("rtop %s starting up", VERSION)
	Debug("Command line arguments: host=%s, port=%d, username=%s, key=%s, interval=%v",
		host, port, username, key, interval)

	// get current user
	var err error
	currentUser, err = user.Current()
	if err != nil {
		Fatal("Failed to get current user: %v", err)
		return
	}
	Debug("Current user: %s", currentUser.Username)

	// fill from ~/.ssh/config if possible
	sshConfig := filepath.Join(currentUser.HomeDir, ".ssh", "config")
	if _, err := os.Stat(sshConfig); err == nil {
		Debug("Found SSH config at %s", sshConfig)
		if parseSshConfig(sshConfig) {
			Debug("Successfully parsed SSH config")
			shost, sport, suser, skey := getSshEntry(host)
			if len(shost) > 0 {
				Debug("Using host from SSH config: %s", shost)
				host = shost
			}
			if sport != 0 && port == 0 {
				Debug("Using port from SSH config: %d", sport)
				port = sport
			}
			if len(suser) > 0 && len(username) == 0 {
				Debug("Using username from SSH config: %s", suser)
				username = suser
			}
			if len(skey) > 0 && len(key) == 0 {
				Debug("Using key from SSH config: %s", skey)
				key = skey
			}
		} else {
			Debug("Failed to parse SSH config")
		}
	} else {
		Debug("SSH config not found at %s", sshConfig)
	}

	// fill in still-unknown ones with defaults
	if port == 0 {
		Debug("Using default port: 22")
		port = 22
	}
	if len(username) == 0 {
		Debug("Using current username: %s", currentUser.Username)
		username = currentUser.Username
	}
	if len(key) == 0 {
		idrsap := filepath.Join(currentUser.HomeDir, ".ssh", "id_rsa")
		if _, err := os.Stat(idrsap); err == nil {
			Debug("Using default SSH key: %s", idrsap)
			key = idrsap
		} else {
			Debug("Default SSH key not found at %s", idrsap)
		}
	}
	if interval == 0 {
		Debug("Using default refresh interval: %d seconds", DEFAULT_REFRESH)
		interval = DEFAULT_REFRESH * time.Second
	}

	Info("Connecting to %s@%s:%d using key %s", username, host, port, key)
	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := sshConnect(username, addr, key)
	if err != nil {
		Fatal("SSH connect error: %v", err)
		os.Exit(2)
	}
	Info("Successfully connected to %s", addr)

	validateOS(client)

	output := getOutput()
	// the loop
	Info("Starting monitoring loop with refresh interval of %v", interval)
	showStats(output, client)
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	timer := time.Tick(interval)
	done := false
	for !done {
		select {
		case s := <-sig:
			Info("Received signal %v, shutting down", s)
			done = true
			fmt.Println()
		case <-timer:
			Debug("Refreshing statistics")
			showStats(output, client)
		}
	}

	Info("rtop shutting down")
	// Close any open resources
	if rtopLogger != nil {
		rtopLogger.Close()
	}
}

func showStats(output io.Writer, client *ssh.Client) {
	stats := Stats{}

	Debug("Collecting system statistics")
	getAllStats(client, &stats)

	// Log summary of collected stats
	used := stats.MemTotal - stats.MemFree - stats.MemBuffers - stats.MemCached
	memUsedPercent := float64(used) / float64(stats.MemTotal) * 100
	cpuIdlePercent := stats.CPU.Idle
	rtopLogger.LogStats(stats.Hostname, stats.Load1, memUsedPercent, float64(cpuIdlePercent))

	clearConsole()
	used = stats.MemTotal - stats.MemFree - stats.MemBuffers - stats.MemCached
	fmt.Fprintf(output,
		`%s%s%s%s up %s%s%s

Load:
    %s%s %s %s%s

CPU:
    %s%.2f%s%% user, %s%.2f%s%% sys, %s%.2f%s%% nice, %s%.2f%s%% idle, %s%.2f%s%% iowait, %s%.2f%s%% hardirq, %s%.2f%s%% softirq, %s%.2f%s%% guest

Processes:
    %s%s%s running of %s%s%s total

Memory:
    free    = %s%s%s
    used    = %s%s%s
    buffers = %s%s%s
    cached  = %s%s%s
    swap    = %s%s%s free of %s%s%s

`,
		escClear,
		escBrightWhite, stats.Hostname, escReset,
		escBrightWhite, fmtUptime(&stats), escReset,
		escBrightWhite, stats.Load1, stats.Load5, stats.Load10, escReset,
		escBrightWhite, stats.CPU.User, escReset,
		escBrightWhite, stats.CPU.System, escReset,
		escBrightWhite, stats.CPU.Nice, escReset,
		escBrightWhite, stats.CPU.Idle, escReset,
		escBrightWhite, stats.CPU.Iowait, escReset,
		escBrightWhite, stats.CPU.Irq, escReset,
		escBrightWhite, stats.CPU.SoftIrq, escReset,
		escBrightWhite, stats.CPU.Guest, escReset,
		escBrightWhite, stats.RunningProcs, escReset,
		escBrightWhite, stats.TotalProcs, escReset,
		escBrightWhite, fmtBytes(stats.MemFree), escReset,
		escBrightWhite, fmtBytes(used), escReset,
		escBrightWhite, fmtBytes(stats.MemBuffers), escReset,
		escBrightWhite, fmtBytes(stats.MemCached), escReset,
		escBrightWhite, fmtBytes(stats.SwapFree), escReset,
		escBrightWhite, fmtBytes(stats.SwapTotal), escReset,
	)
	if len(stats.FSInfos) > 0 {
		fmt.Println("Filesystems:")
		for _, fs := range stats.FSInfos {
			fmt.Fprintf(output, "%s%28s%s    %s%8s%s: %s%s%s free of %s%s%s\n",
				escBrightWhite, fs.Device, escReset,
				escBrightWhite, fs.MountPoint, escReset,
				escBrightWhite, fmtBytes(fs.Free), escReset,
				escBrightWhite, fmtBytes(fs.Used+fs.Free), escReset,
			)
		}
		fmt.Println()
	}
	if len(stats.NetIntf) > 0 {
		fmt.Println("Network Interfaces:")
		keys := make([]string, 0, len(stats.NetIntf))
		for intf := range stats.NetIntf {
			keys = append(keys, intf)
		}
		sort.Strings(keys)
		for _, intf := range keys {
			info := stats.NetIntf[intf]
			fmt.Fprintf(output, "    %s%s%s - %s%s%s",
				escBrightWhite, intf, escReset,
				escBrightWhite, info.IPv4, escReset,
			)
			if len(info.IPv6) > 0 {
				fmt.Fprintf(output, ", %s%s%s\n",
					escBrightWhite, info.IPv6, escReset,
				)
			} else {
				fmt.Fprintf(output, "\n")
			}
			fmt.Fprintf(output, "      rx = %s%s%s, tx = %s%s%s\n",
				escBrightWhite, fmtBytes(info.Rx), escReset,
				escBrightWhite, fmtBytes(info.Tx), escReset,
			)
			fmt.Println()
		}
		fmt.Println()
	}
}

const (
	escClear       = "\033[H\033[2J"
	escRed         = "\033[31m"
	escReset       = "\033[0m"
	escBrightWhite = "\033[37;1m"
)
