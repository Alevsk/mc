/*
 * Minio Client (C) 2014, 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"time"

	"github.com/minio/cli"
	"github.com/minio/mc/pkg/console"
	"github.com/minio/minio/pkg/probe"
	"github.com/minio/pb"
)

// Check for the environment early on and gracefully report.
func checkConfig() {
	_, e := user.Current()
	fatalIf(probe.NewError(e), "Unable to determine current user.")

	// Ensures config file is sane
	_, err := getMcConfig()
	fatalIf(err.Trace(), "Unable to access configuration file.")
}

func migrate() {
	// Migrate config files if any.
	migrateConfig()
	// Migrate session files if any.
	migrateSession()
}

// Get os/arch/platform specific information.
// Returns a map of current os/arch/platform/memstats
func getSystemData() map[string]string {
	host, e := os.Hostname()
	fatalIf(probe.NewError(e), "Unable to determine the hostname.")

	memstats := &runtime.MemStats{}
	runtime.ReadMemStats(memstats)
	mem := fmt.Sprintf("Used: %s | Allocated: %s | UsedHeap: %s | AllocatedHeap: %s",
		pb.FormatBytes(int64(memstats.Alloc)),
		pb.FormatBytes(int64(memstats.TotalAlloc)),
		pb.FormatBytes(int64(memstats.HeapAlloc)),
		pb.FormatBytes(int64(memstats.HeapSys)))
	platform := fmt.Sprintf("Host: %s | OS: %s | Arch: %s", host, runtime.GOOS, runtime.GOARCH)
	goruntime := fmt.Sprintf("Version: %s | CPUs: %s", runtime.Version(), strconv.Itoa(runtime.NumCPU()))
	return map[string]string{
		"PLATFORM": platform,
		"RUNTIME":  goruntime,
		"MEM":      mem,
	}
}

func registerBefore(ctx *cli.Context) error {
	setMcConfigDir(ctx.GlobalString("config-folder"))
	globalQuietFlag = ctx.GlobalBool("quiet")
	globalMimicFlag = ctx.GlobalBool("mimic")
	globalDebugFlag = ctx.GlobalBool("debug")
	globalJSONFlag = ctx.GlobalBool("json")
	if globalDebugFlag {
		console.NoDebugPrint = false
	}

	verifyMCRuntime()

	// Migrate any old version of config / state files to newer format.
	migrate()

	checkConfig()
	return nil
}

// getFormattedVersion -
func getFormattedVersion() string {
	t, _ := time.Parse(time.RFC3339Nano, Version)
	if t.IsZero() {
		return ""
	}
	return t.Format(http.TimeFormat)
}

func registerApp() *cli.App {
	// Register all the commands
	registerCmd(lsCmd)      // List contents of a bucket
	registerCmd(mbCmd)      // make a bucket
	registerCmd(catCmd)     // concantenate an object to standard output
	registerCmd(cpCmd)      // copy objects and files from multiple sources to single destination
	registerCmd(mirrorCmd)  // mirror objects and files from single source to multiple destinations
	registerCmd(sessionCmd) // session handling for resuming copy and mirror operations
	registerCmd(shareCmd)   // share any given url for third party access
	registerCmd(diffCmd)    // compare two objects
	registerCmd(accessCmd)  // set permissions [public, private, readonly, authenticated] for buckets and folders.
	registerCmd(configCmd)  // generate configuration "/home/harsha/.mc/config.json" file.
	registerCmd(updateCmd)  // update Check for new software updates
	registerCmd(versionCmd) // print version

	// register all the flags
	registerFlag(configFlag) // path to config folder
	registerFlag(quietFlag)  // suppress console output
	registerFlag(mimicFlag)  // OS toolchain mimic
	registerFlag(jsonFlag)   // json formatted output
	registerFlag(debugFlag)  // enable debugging output

	app := cli.NewApp()
	app.Usage = "Minio Client for cloud storage and filesystems"
	// hide --version flag, version is a command
	app.HideVersion = true
	app.Commands = commands
	app.Flags = flags
	app.Author = "Minio.io"
	app.CustomAppHelpTemplate = `NAME:
  {{.Name}} - {{.Usage}}

USAGE:
  {{.Name}} {{if .Flags}}[global flags] {{end}}command{{if .Flags}} [command flags]{{end}} [arguments...]

COMMANDS:
  {{range .Commands}}{{join .Names ", "}}{{ "\t" }}{{.Usage}}
  {{end}}{{if .Flags}}
GLOBAL FLAGS:
  {{range .Flags}}{{.}}
  {{end}}{{end}}
VERSION:

` + getFormattedVersion() +
		`
  {{range $key, $value := ExtraInfo}}
{{$key}}:
  {{$value}}
{{end}}
`
	app.CommandNotFound = func(ctx *cli.Context, command string) {
		fatalIf(probe.NewError(errors.New("")), fmt.Sprintf("Command not found: ‘%s’", command))
	}
	return app
}

func main() {
	app := registerApp()
	app.Before = registerBefore

	app.ExtraInfo = func() map[string]string {
		if globalDebugFlag {
			return getSystemData()
		}
		return make(map[string]string)
	}

	app.RunAndExitOnError()
}
