# Running osg ui as a system service

`osg ui` is the local web dashboard. By default it runs in the
foreground of whatever shell launched it — convenient for development,
but useless for the auto-publish workflow (drafts with `osg.publish_at`)
because the scheduler service stops the moment you close the terminal
or reboot.

This page covers the two pieces that turn the dashboard into a
"set-and-forget" daemon: a config flag that auto-starts the
background services every time `osg ui` boots, and a CLI subcommand
that installs `osg ui` itself as a user-level system service so it
arrives back at login and restarts on crash.

## Auto-start services inside `osg ui`

The dashboard knows about four long-running services: `serve`, `api`,
`watcher` and `scheduler`. Until now you had to start each one by
hand from the `/services` page after every dashboard restart. The
`ui.autostart` config key triggers them automatically on boot:

```yaml
# config.yaml
ui:
  addr: ":1314"
  autostart:
    - scheduler   # required for auto-publish (osg.publish_at) to fire
    - watcher     # auto-rebuild on vault changes
    # - serve     # uncomment if you want osg serve always running
    # - api       # uncomment if you use interactions / view counters
```

Restart `osg ui` and you'll see one info line per started service in
the logs: `ui.autostart: service started name=scheduler`. The runner
rejects double-starts, so re-runs are safe; unknown service names log
a warning instead of failing.

If you want **none** to auto-start (the historical default), omit the
field or leave the list empty.

> **Tip:** start with just `scheduler` if your only goal is keeping
> the auto-publish flow alive. `watcher` adds CPU/disk load every
> time you save a file in the vault — only enable it if you also use
> `osg serve` for a live-reload preview.

## Install `osg ui` as a system service

`osg service install` writes a unit file that the OS uses to manage
the dashboard for you: arranca al login, reinicia ante crashes,
sobrevive a reboots.

### macOS — LaunchAgent

```bash
osg service install
```

Writes `~/Library/LaunchAgents/com.jllopis.osg-ui.plist` and loads
it via `launchctl bootstrap gui/<uid>`. The plist sets:

| Key            | Value         | Effect                                     |
|----------------|---------------|--------------------------------------------|
| `RunAtLoad`    | `true`        | Starts immediately when bootstrapped       |
| `KeepAlive`    | `true`        | macOS restarts the agent after any exit    |
| `WorkingDirectory` | install cwd | Resolves `config.yaml` like a manual run |

No `sudo` is involved — LaunchAgents live in the user's home and
load automatically when that user logs in. If you want the dashboard
to come up before login (e.g. on a headless Mac mini), see
*Customising the install* below.

Verify with:

```bash
launchctl list | grep com.jllopis
osg service status      # delegates to `launchctl print gui/<uid>/com.jllopis.osg-ui`
tail -f .osg/logs/osg-ui.out.log
```

### Linux — systemd user service

```bash
osg service install
```

Writes `~/.config/systemd/user/osg-ui.service` and runs
`systemctl --user daemon-reload && systemctl --user enable --now osg-ui`.
The unit declares:

| Directive          | Value              | Effect                              |
|--------------------|--------------------|-------------------------------------|
| `Restart=`         | `on-failure`       | Restarts only on non-zero exit      |
| `RestartSec=`      | `5`                | Waits 5s before restarting          |
| `WantedBy=`        | `default.target`   | Starts at user login                |

Verify with:

```bash
systemctl --user status osg-ui
journalctl --user -u osg-ui -f      # if you change StandardOutput= to journal
tail -f .osg/logs/osg-ui.out.log    # default file-based logs
```

To make the service start at boot (before login), enable lingering:

```bash
loginctl enable-linger $USER
```

### What happens on install

Both platforms share the same prep step:

1. Resolve the absolute path of the current `osg` binary
   (`os.Executable()`), the working directory (cwd) and the config
   path (the global `-c` flag, or `config.yaml`).
2. Create `<workdir>/.osg/logs/` if missing.
3. Write the platform-specific unit file with those absolute paths.
4. Hand the unit to systemd / launchd and start it (unless
   `--no-start` is passed).

Re-running `osg service install` is idempotent — the previous
instance is unloaded, the file rewritten, and the new instance
loaded. Useful when you've moved the binary or changed the working
directory.

## Day-to-day commands

```bash
osg service start       # start the (already-installed) service
osg service stop        # stop it (the unit file stays on disk)
osg service status      # platform-native status output
osg service uninstall   # stop, disable, and remove the unit file
```

`status` defers to `systemctl --user status` or `launchctl print`
without altering exit codes — read the native output for diagnostics.

## Customising the install

| Flag         | Default                    | When you'd change it                          |
|--------------|----------------------------|-----------------------------------------------|
| `--workdir`  | current shell directory    | Service should run as if invoked from another folder (`/var/www/blog`, etc.) |
| `--config`   | global `-c` value or `config.yaml` (joined to workdir if relative) | Keep multiple sites on one machine, each with its own config |
| `--exec`     | `os.Executable()`          | The binary lives somewhere different from where you ran install (e.g. you ran from a build dir but want the unit to point at `/usr/local/bin/osg`) |
| `--no-start` | unset                      | Inspect the generated unit before letting it run |

Example for a system-wide setup (Linux):

```bash
osg service install \
  --workdir /var/www/blog \
  --config /etc/osg/config.yaml \
  --exec /usr/local/bin/osg
```

## Logs

By default both platforms write file-based logs to
`<workdir>/.osg/logs/osg-ui.out.log` and `osg-ui.err.log`. These
rotate only via the OS's log management (none by default) — keep an
eye on them or wire `logrotate` / `newsyslog` if the dashboard runs
for months.

If you prefer systemd's journal on Linux, edit the unit file after
install:

```ini
StandardOutput=journal
StandardError=journal
```

Then `systemctl --user daemon-reload && systemctl --user restart osg-ui`.

## Uninstall

```bash
osg service uninstall
```

Stops the service (idempotent: silently skips if not running),
deletes the unit file, and reloads systemd / launchd so the unit is
forgotten. The config.yaml, content/, public/ and `.osg/cache/`
directories are untouched.

## Troubleshooting

**macOS: "Service is being throttled" after re-installing**
launchd protects against crash loops. Wait ~10 seconds and re-run
`osg service install` (or `osg service start`). Check
`/var/log/system.log` if it persists.

**macOS: agent never starts at login on a headless Mac**
LaunchAgents only load when a GUI session exists. Either log in
once (the agent stays loaded across reboots if `loginwindow`
re-runs you) or convert to a LaunchDaemon — file an issue / patch
if you need first-class support for that scope.

**Linux: `Failed to connect to bus` when running `osg service install`**
You're not in a real user session (e.g. inside a basic SSH login).
Run `loginctl enable-linger $USER` once, then re-run install.

**The service starts but `/scheduler` is empty after restart**
Check `ui.autostart` includes `scheduler`. The dashboard process
running as the service still needs the config flag — without it, you
have a dashboard but no scheduler watching for `publish_at` dates.

**Logs say "address already in use"**
Another instance of `osg ui` is already on `:1314` (maybe a leftover
manual run). Stop the manual process or change `ui.addr` and reinstall.

## Unsupported platforms

Windows, FreeBSD and other Unix-like systems return a clear
"`osg service: <os> is not a supported platform`" error. You can
still run `osg ui` manually or wire it into the platform's native
service manager (Task Scheduler, rc.d) — open an issue if you'd
like first-class support added.
