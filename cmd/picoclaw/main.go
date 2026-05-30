// Command picoclaw is the CLI entry point for the PicoClaw runtime.
//
// Phase 1 subcommands:
//
//	picoclaw version
//	picoclaw onboard               # write a starter config.json
//	picoclaw status                # show config summary
//	picoclaw agent -m "..."        # one-shot agent run
//	picoclaw agent                 # interactive REPL
//	picoclaw gateway               # start HTTP gateway + enabled channels
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"picoclaw/pkg/agent"
	"picoclaw/pkg/channels"
	"picoclaw/pkg/config"
	"picoclaw/pkg/gateway"
	websrv "picoclaw/web/server"
)

const version = "0.1.0-phase2"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	var err error
	switch cmd {
	case "version", "-v", "--version":
		fmt.Println("picoclaw", version)
	case "onboard":
		err = cmdOnboard(args)
	case "status":
		err = cmdStatus(args)
	case "agent":
		err = cmdAgent(args)
	case "gateway":
		err = cmdGateway(args)
	case "web":
		err = cmdWeb(args)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`picoclaw — ultra-lightweight personal AI assistant

usage:
  picoclaw version
  picoclaw onboard [-config config.json]
  picoclaw status  [-config config.json]
  picoclaw agent   [-config config.json] [-m "message"]
  picoclaw gateway [-config config.json]
  picoclaw web     [-config config.json] [-addr 127.0.0.1:18800] [-public]
`)
}

// configFlag adds a shared -config flag to a FlagSet.
func configFlag(fs *flag.FlagSet) *string {
	return fs.String("config", "config.json", "path to config.json")
}

func cmdOnboard(args []string) error {
	fs := flag.NewFlagSet("onboard", flag.ExitOnError)
	path := configFlag(fs)
	_ = fs.Parse(args)

	if _, err := os.Stat(*path); err == nil {
		return fmt.Errorf("%s already exists; refusing to overwrite", *path)
	}
	if err := os.WriteFile(*path, []byte(starterConfig), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s — set an api_key and run: picoclaw agent -m \"hello\"\n", *path)
	return nil
}

func cmdStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	path := configFlag(fs)
	_ = fs.Parse(args)

	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	fmt.Println("picoclaw", version)
	fmt.Println("workspace:", cfg.Workspace)
	fmt.Println("default model:", cfg.Agents.Defaults.ModelName)
	fmt.Println("gateway:", cfg.Gateway.Addr())
	fmt.Printf("models (%d):\n", len(cfg.ModelList))
	for _, m := range cfg.ModelList {
		fmt.Printf("  - %s\n", m.Name)
	}
	fmt.Printf("telegram enabled: %v\n", cfg.Channels.Telegram.Enabled)
	return nil
}

func cmdAgent(args []string) error {
	fs := flag.NewFlagSet("agent", flag.ExitOnError)
	path := configFlag(fs)
	msg := fs.String("m", "", "one-shot message (omit for interactive REPL)")
	_ = fs.Parse(args)

	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	ag, err := agent.New(cfg)
	if err != nil {
		return err
	}
	ag.Observer = &cliObserver{}

	ctx := signalContext()

	if *msg != "" {
		reply, err := ag.Run(ctx, *msg)
		if err != nil {
			return err
		}
		fmt.Println(reply)
		return nil
	}

	// Interactive REPL.
	fmt.Println("picoclaw agent — type a message, or 'exit' to quit")
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for {
		fmt.Print("> ")
		if !sc.Scan() {
			return nil
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			return nil
		}
		reply, err := ag.Run(ctx, line)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			continue
		}
		fmt.Println(reply)
	}
}

func cmdGateway(args []string) error {
	fs := flag.NewFlagSet("gateway", flag.ExitOnError)
	path := configFlag(fs)
	_ = fs.Parse(args)

	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	ag, err := agent.New(cfg)
	if err != nil {
		return err
	}

	ctx := signalContext()

	// Start the cron scheduler so scheduled prompts fire.
	go func() { _ = ag.Scheduler().Run(ctx) }()

	// Start the Telegram channel if enabled.
	if cfg.Channels.Telegram.Enabled && cfg.Channels.Telegram.Token != "" {
		tg := channels.NewTelegram(cfg.Channels.Telegram.Token)
		go func() {
			handle := func(c context.Context, user, text string) (string, error) {
				return ag.Run(c, text)
			}
			if err := tg.Run(ctx, handle); err != nil && ctx.Err() == nil {
				fmt.Fprintln(os.Stderr, "telegram:", err)
			}
		}()
		fmt.Println("telegram channel started")
	}

	gw := gateway.New(cfg.Gateway.Addr(), ag)
	return gw.Run(ctx)
}

func cmdWeb(args []string) error {
	fs := flag.NewFlagSet("web", flag.ExitOnError)
	path := configFlag(fs)
	public := fs.Bool("public", false, "bind on all interfaces instead of loopback")
	addr := fs.String("addr", "", "listen address (default 127.0.0.1:18800, or 0.0.0.0:18800 with -public)")
	_ = fs.Parse(args)

	cfg, err := config.Load(*path)
	if err != nil {
		return err
	}
	listen := *addr
	if listen == "" {
		if *public {
			listen = "0.0.0.0:18800"
		} else {
			listen = "127.0.0.1:18800"
		}
	}
	l, err := websrv.New(listen, *public, cfg)
	if err != nil {
		return err
	}
	return l.Run(signalContext())
}

// signalContext returns a context cancelled on SIGINT/SIGTERM.
func signalContext() context.Context {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	_ = stop // cancelled implicitly on process exit
	return ctx
}

// cliObserver prints loop activity to stderr so the user sees tool use.
type cliObserver struct{}

func (cliObserver) OnAssistant(string) {}
func (cliObserver) OnToolCall(name, args string) {
	fmt.Fprintf(os.Stderr, "  · %s %s\n", name, truncate(args, 120))
}
func (cliObserver) OnToolResult(name, result string) {
	fmt.Fprintf(os.Stderr, "  ↳ %s\n", truncate(strings.ReplaceAll(result, "\n", " "), 120))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

const starterConfig = `{
  "version": 1,
  "workspace": "./workspace",
  "model_list": [
    {
      "name": "openai/gpt-4o-mini",
      "api_key": "sk-REPLACE_ME"
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "openai/gpt-4o-mini",
      "max_turns": 12
    }
  },
  "channels": {
    "telegram": {
      "enabled": false,
      "token": ""
    }
  },
  "gateway": {
    "host": "127.0.0.1",
    "port": 18790
  }
}
`
