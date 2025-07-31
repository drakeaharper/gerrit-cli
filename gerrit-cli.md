# Gerrit CLI Tool Documentation

A command-line interface for interacting with Gerrit Code Review, specifically designed for developers who prefer terminal workflows over web UIs.

## Overview

This CLI tool provides easy access to common Gerrit operations including:
- Viewing open changes
- Reading review comments
- Checking build status
- Managing changes locally

## Prerequisites

- SSH access to Gerrit server
- HTTP password for REST API access (found in Gerrit Settings → HTTP Password)
- `jq` installed for JSON parsing
- Git configured with Gerrit remote

## Configuration

### Environment Variables

Add these to your `~/.zshrc` or `~/.bashrc`:

```bash
export GERRIT_SERVER="gerrit.company.com"
export GERRIT_PORT="29418"
export GERRIT_USER="your.username"
export GERRIT_HTTP="your-http-password"
export GERRIT_PROJECT="your-project"
```

### SSH Config (Optional)

Add to `~/.ssh/config` for easier SSH access:

```
Host gerrit
    HostName gerrit.company.com
    Port 29418
    User your.username
```

## Basic Commands

### List Your Open Changes

```bash
# Simple list
ssh -p $GERRIT_PORT $GERRIT_USER@$GERRIT_SERVER gerrit query \
  --format=JSON status:open owner:self | \
  grep -v '"type":"stats"' | \
  jq -r '"\(.number) - \(.subject)"'

# Detailed view with review status
ssh -p $GERRIT_PORT $GERRIT_USER@$GERRIT_SERVER gerrit query \
  --format=JSON --current-patch-set status:open owner:self | \
  grep -v '"type":"stats"' | \
  jq -r '"\(.number) - \(.subject) (PS: \(.currentPatchSet.number))"'
```

### View Comments on a Change

```bash
# Get all comments (including resolved)
change_id="384465"
curl -s -u "$GERRIT_USER:$GERRIT_HTTP" \
  "https://$GERRIT_SERVER/a/changes/$change_id/comments" | \
  sed '1s/^)]}\x27//' | jq .

# Get only unresolved comments
curl -s -u "$GERRIT_USER:$GERRIT_HTTP" \
  "https://$GERRIT_SERVER/a/changes/$change_id/comments" | \
  sed '1s/^)]}\x27//' | \
  jq -r 'to_entries[] | select(.key != "/PATCHSET_LEVEL") | 
    "\n📄 \(.key)\n" + 
    (.value[] | select(.unresolved == true) | 
    "   Line \(.line): \(.message)\n   Author: \(.author.name)\n")'
```

### Check Change Details

```bash
# Get change info via SSH
ssh -p $GERRIT_PORT $GERRIT_USER@$GERRIT_SERVER gerrit query \
  --format=JSON --current-patch-set --files change:384465

# Get change info via REST API
curl -s -u "$GERRIT_USER:$GERRIT_HTTP" \
  "https://$GERRIT_SERVER/a/changes/384465/detail" | \
  sed '1s/^)]}\x27//' | jq .
```

### Work with Changes Locally

```bash
# Fetch and cherry-pick a change
change_num="384465"
patchset="8"
ref_num="${change_num: -2}"  # Last 2 digits

git fetch ssh://$GERRIT_USER@$GERRIT_SERVER:$GERRIT_PORT/$GERRIT_PROJECT \
  refs/changes/$ref_num/$change_num/$patchset && \
git cherry-pick FETCH_HEAD

# Or use the shorter format if you have the full ref
git fetch origin refs/changes/65/384465/8 && git cherry-pick FETCH_HEAD
```

## Useful Aliases

Add these to your shell configuration:

```bash
# List my open changes
alias gmine='ssh -p $GERRIT_PORT $GERRIT_USER@$GERRIT_SERVER gerrit query \
  --format=JSON status:open owner:self | \
  grep -v "type.*stats" | \
  jq -r "\"\\(.number) - \\(.subject)\""'

# Get unresolved comments on a change
gcomments() {
  local change_id="$1"
  curl -s -u "$GERRIT_USER:$GERRIT_HTTP" \
    "https://$GERRIT_SERVER/a/changes/$change_id/comments" | \
    sed '1s/^)]}\x27//' | \
    jq -r 'to_entries[] | select(.key != "/PATCHSET_LEVEL") | 
      "\n📄 \(.key)\n" + 
      (.value[] | select(.unresolved == true) | 
      "   Line \(.line): \(.message)\n   By: \(.author.name)\n")'
}

# Fetch and checkout a change
gfetch() {
  local change_num="$1"
  local patchset="${2:-current}"
  
  if [ "$patchset" = "current" ]; then
    # Get current patchset number
    patchset=$(ssh -p $GERRIT_PORT $GERRIT_USER@$GERRIT_SERVER gerrit query \
      --format=JSON --current-patch-set change:$change_num | \
      grep -v "type.*stats" | \
      jq -r '.currentPatchSet.number')
  fi
  
  local ref_num="${change_num: -2}"
  git fetch ssh://$GERRIT_USER@$GERRIT_SERVER:$GERRIT_PORT/$GERRIT_PROJECT \
    refs/changes/$ref_num/$change_num/$patchset && \
  git checkout FETCH_HEAD
}

# Cherry-pick a change
gcherry() {
  local change_num="$1"
  local patchset="${2:-current}"
  
  if [ "$patchset" = "current" ]; then
    patchset=$(ssh -p $GERRIT_PORT $GERRIT_USER@$GERRIT_SERVER gerrit query \
      --format=JSON --current-patch-set change:$change_num | \
      grep -v "type.*stats" | \
      jq -r '.currentPatchSet.number')
  fi
  
  local ref_num="${change_num: -2}"
  git fetch ssh://$GERRIT_USER@$GERRIT_SERVER:$GERRIT_PORT/$GERRIT_PROJECT \
    refs/changes/$ref_num/$change_num/$patchset && \
  git cherry-pick FETCH_HEAD
}
```

## Advanced Usage

### Query with Custom Filters

```bash
# Changes needing your review
ssh -p $GERRIT_PORT $GERRIT_USER@$GERRIT_SERVER gerrit query \
  --format=JSON "status:open reviewer:self -owner:self" | \
  jq -r 'select(.type != "stats") | "\(.number) - \(.subject) by \(.owner.name)"'

# Changes in a specific project branch
ssh -p $GERRIT_PORT $GERRIT_USER@$GERRIT_SERVER gerrit query \
  --format=JSON "status:open project:$GERRIT_PROJECT branch:master owner:self"

# Recently updated changes
ssh -p $GERRIT_PORT $GERRIT_USER@$GERRIT_SERVER gerrit query \
  --format=JSON "status:open owner:self -age:7d"
```

### Get Review Scores

```bash
change_id="384465"
ssh -p $GERRIT_PORT $GERRIT_USER@$GERRIT_SERVER gerrit query \
  --format=JSON --current-patch-set --all-approvals change:$change_id | \
  jq '.currentPatchSet.approvals'
```

### Batch Operations

```bash
# Get comments for all your open changes
for change in $(ssh -p $GERRIT_PORT $GERRIT_USER@$GERRIT_SERVER gerrit query \
  --format=JSON status:open owner:self | \
  grep -v "type.*stats" | jq -r '.number'); do
  echo "=== Change $change ==="
  gcomments $change
done
```

## Troubleshooting

### SSH Connection Issues
- Verify SSH key is uploaded to Gerrit (Settings → SSH Keys)
- Test connection: `ssh -p 29418 $GERRIT_USER@$GERRIT_SERVER`
- Check for welcome message

### REST API Authentication
- HTTP password is different from login password
- Generate new password at: https://$GERRIT_SERVER/#/settings/http-password
- Test auth: `curl -s -u "$GERRIT_USER:$GERRIT_HTTP" "https://$GERRIT_SERVER/a/accounts/self"`

### Common Error Messages
- "Not found: 384465" - Use full change identifier format
- "Unauthorized" - Check HTTP password and username
- "parse error" - Response includes XSSI prefix, use `sed '1s/^)]}\x27//'`

## Future Enhancements

1. **Full CLI Tool**: Build a proper CLI tool in Go/Rust that handles:
   - Configuration management
   - Error handling and retries
   - Output formatting options
   - Interactive mode

2. **Additional Features**:
   - Post comments via CLI
   - Submit/abandon changes
   - Rebase changes
   - Dashboard view with change statistics

3. **Integration**:
   - Git hooks for automatic operations
   - Editor plugins (VSCode, Vim)
   - Shell prompt integration

## Contributing

To improve Gerrit's CLI capabilities:
1. Gerrit source: https://gerrit.googlesource.com/gerrit
2. Feature requests: https://bugs.chromium.org/p/gerrit/issues
3. Mailing list: repo-discuss@googlegroups.com

Consider contributing to add inline comment support to the SSH API!