# Branch Protection Setup

To ensure all quality gates are enforced, configure the following branch protection rules for the `main` branch:

## GitHub Branch Protection Rules

### Required Settings

1. **Restrict pushes to matching branches**
   - ✅ Require pull request reviews before merging
   - ✅ Required number of reviewers: 1
   - ✅ Dismiss stale pull request approvals when new commits are pushed

2. **Require status checks to pass before merging**
   - ✅ Require branches to be up to date before merging
   - ✅ Required status checks:
     - `Format Check`
     - `Lint`
     - `Test (ubuntu-latest)`
     - `Test (macos-latest)`  
     - `Test (windows-latest)`
     - `Coverage Check (70% minimum)`
     - `Build`
     - `Performance Benchmarks`
     - `Security Scan`

3. **Additional restrictions**
   - ✅ Require conversation resolution before merging
   - ✅ Include administrators (enforce for everyone)
   - ✅ Allow force pushes: **NO**
   - ✅ Allow deletions: **NO**

### Setup Commands (GitHub CLI)

```bash
# Set up branch protection with all required checks
gh api repos/:owner/:repo/branches/main/protection \
  --method PUT \
  --field required_status_checks='{"strict":true,"checks":[{"context":"Format Check"},{"context":"Lint"},{"context":"Test (ubuntu-latest)"},{"context":"Test (macos-latest)"},{"context":"Test (windows-latest)"},{"context":"Coverage Check (70% minimum)"},{"context":"Build"},{"context":"Performance Benchmarks"},{"context":"Security Scan"}]}' \
  --field enforce_admins=true \
  --field required_pull_request_reviews='{"required_approving_review_count":1,"dismiss_stale_reviews":true}' \
  --field restrictions=null \
  --field required_conversation_resolution=true \
  --field allow_force_pushes=false \
  --field allow_deletions=false
```

### Manual Setup (GitHub Web Interface)

1. Go to repository **Settings** → **Branches**
2. Click **Add rule** for branch `main`
3. Configure all settings as listed above
4. Click **Create** to save the protection rule

## Result

With these settings:
- ✅ No direct pushes to `main` (all changes via PR)
- ✅ All CI checks must pass before merge
- ✅ Code review required from at least 1 person
- ✅ Branch must be up to date with latest `main`
- ✅ All conversations must be resolved
- ✅ Administrators cannot bypass these rules

This ensures that every change to `main` meets all quality gates before being merged.