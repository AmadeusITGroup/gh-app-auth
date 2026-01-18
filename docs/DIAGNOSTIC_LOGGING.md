### PAT Flow Example

PAT flows include `generate_pat_credentials` and `output_pat_credentials` steps and log the configured username (useful for Bitbucket):

```
[2025-10-13T20:02:03.400Z] FLOW_STEP [...] step=match_by_pattern pat_name="Bitbucket PAT" pattern=bitbucket.example.com/
[2025-10-13T20:02:03.401Z] FLOW_STEP [...] step=generate_pat_credentials pat_name="Bitbucket PAT"
[2025-10-13T20:02:03.402Z] FLOW_STEP [...] step=pat_retrieved pat_name="Bitbucket PAT" token_hash=sha256:...
[2025-10-13T20:02:03.403Z] FLOW_STEP [...] step=output_pat_credentials username=your-username token_hash=sha256:...
```

# Diagnostic Logging

The `gh-app-auth` extension includes comprehensive diagnostic logging to help debug git credential flows. This logging is designed to be non-intrusive and secure.

## Features

- **Conditional Activation**: Only logs when explicitly enabled
- **File-based Output**: Logs to file, never interferes with git stdout/stderr
- **Secure by Default**: Never logs tokens in clear text
- **Flow Tracking**: Session and operation IDs for tracing
- **Structured Output**: Easy to parse and build flow diagrams

## Quick Start

### Enable Logging

```bash
# Enable with default log location
export GH_APP_AUTH_DEBUG_LOG=1

# Or specify custom log file
export GH_APP_AUTH_DEBUG_LOG="/path/to/debug.log"

# Now run git operations
git clone https://github.com/myorg/private-repo
```

### View Logs

```bash
# Default log location
tail -f ~/.config/gh/extensions/gh-app-auth/debug.log

# Custom location
tail -f /path/to/debug.log
```

## Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `GH_APP_AUTH_DEBUG_LOG` | Enable logging and optionally set file path | `1` or `/tmp/debug.log` |

## Log File Locations

### Default Location

```
~/.config/gh/extensions/gh-app-auth/debug.log
```

### Fallback Location

If the default location is not writable:

```
/tmp/gh-app-auth-debug.log
```

### Custom Location

Set via environment variable:

```bash
export GH_APP_AUTH_DEBUG_LOG="/var/log/gh-app-auth.log"
```

## Log Format

Each log entry follows this format:

```
[TIMESTAMP] EVENT [OPERATION_ID] key=value key=value...
```

### Example Entry

```
[2024-10-13T19:52:15.123Z] FLOW_STEP [session_1728844335_1234_op5] step=app_matched app_id=123456 app_name="My App" patterns=["github.com/myorg/*"]
```

### Components

- **TIMESTAMP**: ISO 8601 format with milliseconds
- **EVENT**: Type of log event (see Event Types below)
- **OPERATION_ID**: Unique identifier for tracing operations
- **KEY=VALUE**: Structured data fields

## Event Types

### Session Events

| Event | Description |
|-------|-------------|
| `SESSION_START` | New session started |
| `SESSION_END` | Session ended |

### Flow Events

| Event | Description |
|-------|-------------|
| `FLOW_START` | Start of a credential operation |
| `FLOW_STEP` | Step within a flow (e.g., `match_by_pattern`, `generate_credentials`, `generate_pat_credentials`, `output_pat_credentials`) |
| `FLOW_SUCCESS` | Successful completion |
| `FLOW_ERROR` | Error in flow |

### Debug Events

| Event | Description |
|-------|-------------|
| `DEBUG` | General debug information |
| `INFO` | Informational messages |
| `ERROR` | Error messages |

## Security Features

### Token Protection

Tokens are never logged in clear text. Instead, they are hashed:

```
token_hash=sha256:a1b2c3d4e5f6a1b2
```

This allows you to:

- Verify if the same token was generated multiple times
- Identify token-related issues without exposing secrets
- Correlate operations using the same token

### URL Sanitization

URLs with embedded credentials are sanitized:

```bash
# Original: https://user:token@github.com/repo
# Logged:   https://<credentials>@github.com/repo
```

### Configuration Sanitization

Sensitive configuration fields are automatically redacted:

```
private_key=sha256:f7e8d9c0b1a2f7e8
token=sha256:9c8b7a6d5e4f9c8b
password=<redacted>
```

## Git Credential Flow Tracing

### Complete Flow Example

Here's what a successful git credential flow looks like:

```
[2024-10-13T19:52:15.123Z] SESSION_START [session_1728844335_1234_op1] pid=1234 args=["gh-app-auth","git-credential","get"]
[2024-10-13T19:52:15.124Z] FLOW_START [session_1728844335_1234_op2] operation=get flow=START
[2024-10-13T19:52:15.125Z] FLOW_STEP [session_1728844335_1234_op3] step=read_input
[2024-10-13T19:52:15.126Z] FLOW_STEP [session_1728844335_1234_op4] step=parse_input protocol=https host=github.com path=myorg/repo
[2024-10-13T19:52:15.125Z] FLOW_STEP [session_1728844335_1234_op5] step=build_url url=https://github.com/myorg/repo
[2024-10-13T19:52:15.128Z] FLOW_STEP [session_1728844335_1234_op6] step=load_config
[2024-10-13T19:52:15.130Z] FLOW_STEP [session_1728844335_1234_op7] step=config_loaded app_count=1
[2024-10-13T19:52:15.131Z] FLOW_STEP [session_1728844335_1234_op8] step=match_app url=https://github.com/myorg/repo
[2024-10-13T19:52:15.132Z] FLOW_STEP [session_1728844335_1234_op9] step=app_matched app_id=123456 app_name="My App"
[2024-10-13T19:52:15.133Z] FLOW_STEP [session_1728844335_1234_op10] step=generate_credentials app_id=123456
[2024-10-13T19:52:15.456Z] FLOW_STEP [session_1728844335_1234_op11] step=credentials_generated token_hash=sha256:a1b2c3d4e5f6a1b2
[2024-10-13T19:52:15.457Z] FLOW_STEP [session_1728844335_1234_op12] step=output_credentials username="My App[bot]" token_hash=sha256:a1b2c3d4e5f6a1b2
[2024-10-13T19:52:15.458Z] FLOW_SUCCESS [session_1728844335_1234_op13] operation=get flow=SUCCESS
[2024-10-13T19:52:15.459Z] SESSION_END [session_1728844335_1234_op14]
```

### Multi-Stage Protocol

Git often calls the credential helper twice:

**Stage 1: Host Only**

```
[19:52:15.123Z] FLOW_START [session_1234_op1] operation=get
[19:52:15.124Z] FLOW_STEP [session_1234_op2] step=parse_input protocol=https host=github.com
[19:52:15.125Z] FLOW_STEP [session_1234_op3] step=no_path_exit url=https://github.com
[19:52:15.126Z] FLOW_SUCCESS [session_1234_op4] operation=get
```

**Stage 2: Full Path**

```
[19:52:15.200Z] FLOW_START [session_1234_op5] operation=get
[19:52:15.201Z] FLOW_STEP [session_1234_op6] step=parse_input protocol=https host=github.com path=myorg/repo
[19:52:15.202Z] FLOW_STEP [session_1234_op7] step=app_matched app_id=123456
# ... credential generation ...
[19:52:15.300Z] FLOW_SUCCESS [session_1234_op15] operation=get
```

## Common Flow Patterns

### Successful Authentication

1. `FLOW_START` → operation=get
2. `read_input` → Parse git's input
3. `build_url` → Construct repository URL  
4. `load_config` → Load app configuration
5. `match_app` → Find matching app
6. `generate_credentials` → Create JWT and get token
7. `output_credentials` → Return to git
8. `FLOW_SUCCESS` → Complete

### No Configuration

1. `FLOW_START` → operation=get
2. `read_input` → Parse git's input
3. `load_config` → Attempt to load config
4. `no_config` → No apps configured
5. `FLOW_SUCCESS` → Silent exit (allows fallback)

### No Matching App

1. `FLOW_START` → operation=get
2. `read_input` → Parse git's input
3. `build_url` → Construct repository URL
4. `load_config` → Load app configuration
5. `match_app` → Try to find matching app
6. `no_match_exit` → No pattern matches
7. `FLOW_SUCCESS` → Silent exit (allows fallback)

### Authentication Error

1. `FLOW_START` → operation=get
2. ... (steps 2-5 as above)
3. `generate_credentials` → Try to authenticate
4. `FLOW_ERROR` → Authentication failed
5. Error returned to git

## Building Flow Diagrams

### Using grep and awk

Extract flow steps for a specific session:

```bash
grep "session_1728844335_1234" debug.log | \
  awk '{print $3, $4}' | \
  sed 's/\[//g; s/\]//g'
```

### Flow Visualization Script

Create a simple flow tracer:

```bash
#!/bin/bash
# trace-flow.sh - Extract flow for a session

SESSION_ID="$1"
if [ -z "$SESSION_ID" ]; then
    echo "Usage: $0 <session_id>"
    exit 1
fi

grep "$SESSION_ID" ~/.config/gh/extensions/gh-app-auth/debug.log | \
while IFS= read -r line; do
    timestamp=$(echo "$line" | cut -d' ' -f1 | tr -d '[]')
    event=$(echo "$line" | cut -d' ' -f2)
    opid=$(echo "$line" | cut -d' ' -f3 | tr -d '[]')
    data=$(echo "$line" | cut -d' ' -f4-)
    
    echo "$timestamp: $event ($data)"
done
```

### Mermaid Diagram Generation

Convert logs to Mermaid flowchart:

```bash
#!/bin/bash
# generate-mermaid.sh

echo "flowchart TD"
grep "FLOW_" debug.log | \
while IFS= read -r line; do
    step=$(echo "$line" | grep -o 'step=[^[:space:]]*' | cut -d'=' -f2)
    if [ -n "$step" ]; then
        echo "    $step"
    fi
done | \
awk '
BEGIN { prev = "" }
{
    if (prev != "") {
        print "    " prev " --> " $1
    }
    prev = $1
}
'
```

## Troubleshooting

### No Log File Created

**Check:**

1. Environment variable is set: `echo $GH_APP_AUTH_DEBUG_LOG`
2. Directory is writable: `ls -la ~/.config/gh/extensions/gh-app-auth/`
3. Disk space available: `df -h`

**Solution:**

```bash
# Ensure directory exists
mkdir -p ~/.config/gh/extensions/gh-app-auth

# Or use temp directory
export GH_APP_AUTH_DEBUG_LOG="/tmp/gh-app-auth.log"
```

### Empty Log File

**Check:**

1. Git is actually calling the credential helper
2. Operations are completing (may be silent exits)

**Solution:**

```bash
# Force credential helper call
echo -e "protocol=https\nhost=github.com\npath=myorg/repo\n" | \
  gh app-auth git-credential get
```

### Log File Too Large

**Rotate logs:**

```bash
# Archive current log
mv ~/.config/gh/extensions/gh-app-auth/debug.log{,.$(date +%Y%m%d)}

# Start fresh
touch ~/.config/gh/extensions/gh-app-auth/debug.log
```

### Performance Impact

Logging has minimal performance impact:

- Only active when explicitly enabled
- Async file writes
- Structured data (no complex formatting)

To disable:

```bash
unset GH_APP_AUTH_DEBUG_LOG
```

## Integration with CI/CD

### GitHub Actions

```yaml
- name: Debug git credentials
  env:
    GH_APP_AUTH_DEBUG_LOG: "${{ runner.temp }}/gh-app-auth.log"
  run: |
    git clone https://github.com/myorg/private-repo
    
- name: Upload debug logs
  uses: actions/upload-artifact@v3
  if: failure()
  with:
    name: debug-logs
    path: ${{ runner.temp }}/gh-app-auth.log
```

### Local Development

```bash
# Enable logging for development session
export GH_APP_AUTH_DEBUG_LOG="./debug-$(date +%Y%m%d-%H%M).log"

# Run git operations
git clone https://github.com/myorg/private-repo
git pull origin main

# Analyze logs
./trace-flow.sh session_$(date +%s)_$$
```

## Advanced Usage

### Custom Log Analysis

Parse logs to find authentication patterns:

```bash
# Count successful authentications by app
grep "app_matched" debug.log | \
  grep -o 'app_id=[0-9]*' | \
  sort | uniq -c

# Find failed authentications
grep "FLOW_ERROR" debug.log | \
  grep -o 'error=[^[:space:]]*'

# Calculate authentication timing
grep "generate_credentials" debug.log | \
  awk '{print $1}' | \
  tr -d '[]T' | \
  awk -F: '{print $1":"$2":"$3}' | \
  sort | uniq -c
```

### Log Filtering

Extract specific operation types:

```bash
# Only show credential generation
grep -E "(FLOW_START|generate_|FLOW_SUCCESS|FLOW_ERROR)" debug.log

# Show configuration loading issues
grep -E "(load_config|config_loaded|no_config)" debug.log

# Track token usage
grep "token_hash" debug.log | \
  cut -d'=' -f2- | \
  sort | uniq -c
```

## Security Considerations

### Log File Permissions

Default permissions are restrictive (`0600` - owner read/write only):

```bash
ls -la ~/.config/gh/extensions/gh-app-auth/debug.log
# -rw------- 1 user user 1234 Oct 13 19:52 debug.log
```

### Log Rotation

Implement automatic log rotation:

```bash
# Add to crontab
0 0 * * 0 find ~/.config/gh/extensions/gh-app-auth -name "debug.log" -size +10M -exec mv {} {}.$(date +\%Y\%m\%d) \;
```

### Sensitive Data Audit

Even with sanitization, periodically audit logs:

```bash
# Check for potential sensitive data leakage
grep -i -E "(password|secret|key|token)" debug.log | \
  grep -v -E "(token_hash=sha256|private_key_path=|<redacted>)"
```

## Example: Complete Debugging Session

### 1. Enable Logging

```bash
export GH_APP_AUTH_DEBUG_LOG=1
```

### 2. Reproduce Issue

```bash
git clone https://github.com/myorg/failing-repo
```

### 3. Analyze Logs

```bash
# View recent logs
tail -50 ~/.config/gh/extensions/gh-app-auth/debug.log

# Find the session ID
grep "SESSION_START" debug.log | tail -1

# Extract flow for that session
grep "session_1728844335_1234" debug.log
```

### 4. Identify Issue

```bash
# Look for errors
grep "FLOW_ERROR" debug.log | tail -1

# Check if app matching worked
grep "app_matched\|no_match_exit" debug.log | tail -1

# Verify token generation
grep "credentials_generated" debug.log | tail -1
```

### 5. Fix and Verify

```bash
# Make configuration changes
gh app-auth setup --app-id 123456 --patterns "github.com/myorg/*"

# Test again
git clone https://github.com/myorg/failing-repo

# Verify success
grep "FLOW_SUCCESS" debug.log | tail -1
```

This diagnostic logging system provides comprehensive visibility into the git credential flow while maintaining security and performance. Use it to debug authentication issues, verify configuration, and understand git's credential helper protocol interactions.
