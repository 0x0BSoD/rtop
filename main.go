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
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

const VERSION = "1.0"
const DEFAULT_REFRESH = 5 // default refresh interval in seconds

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
	currentUser, err := user.Current()
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

	//var stats Stats
	//getAllStats(client, &stats)
	//
	//fmt.Println("=========================")
	//fmt.Println("=========================")
	//for _, i := range stats.Cgroups {
	//	fmt.Println("Path:", i.Path)
	//	fmt.Println("CPU:", i.CpuUsage)
	//	fmt.Printf("Mem: %d / %d\n", i.MemoryUsageCurrent, i.MemoryUsageLimit)
	//	fmt.Printf("IO: %d / %d\n", i.IoReadBytes, i.IoWriteBytes)
	//	if len(i.Children) != 0 {
	//		fmt.Println("Children:")
	//		printStuff(1, i.Children)
	//	}
	//}
	//fmt.Println("=========================")
	//fmt.Println("=========================")

	Info("Starting monitoring loop with refresh interval of %v", interval)

	// Initialize progress bars
	progressBars := make(map[string]progress.Model, 10)
	progressBars["total"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["system"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["user"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["idle"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["irq"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["softIrq"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["iowait"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["guest"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["nice"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))
	progressBars["steal"] = progress.New(progress.WithScaledGradient("#FF7CCB", "#FDFF8C"))

	m := guiModel{
		client:         client,
		updateInterval: interval,
		bars:           progressBars,
		selected:       -1,
	}
	getAllStats(m.client, &m.stats)
	initFsTable(&m)
	initNetTable(&m)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

	Info("rtop shutting down")
	// Close any open resources
	if rtopLogger != nil {
		rtopLogger.Close()
	}
}

//func printStuff(iter int, cg []*Cgroup) {
//	for _, c := range cg {
//		fmt.Println(strings.Repeat("-", iter), "Path:", c.Path)
//		fmt.Println(strings.Repeat("  ", iter), "CPU:", c.CpuUsage)
//		fmt.Printf("%s Mem: %d / %d\n", strings.Repeat("  ", iter), c.MemoryUsageCurrent, c.MemoryUsageLimit)
//		fmt.Printf("%s IO: %d / %d\n", strings.Repeat("  ", iter), c.IoReadBytes, c.IoWriteBytes)
//		if len(c.Children) != 0 {
//			fmt.Println(strings.Repeat("  ", iter), "Children:")
//			iter += 1
//			printStuff(iter, c.Children)
//		}
//	}
//}
