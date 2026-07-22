- [ ] **Standardize the security workflow to the ghcommon standard** (low
      priority). The standard `security.yml`/`reusable-security.yml` were left
      out of the workflow conversion (#2167) because: (1) the repo has CodeQL
      **default setup** enabled, which GitHub forbids alongside a custom CodeQL
      workflow; (2) the standard job needs a `.github/dependency-review-config.yml`
      this repo lacks; and (3) its `go-audit` step installs a dead module
      (`github.com/securecodewarrior/go-audit`). To adopt it: decide default-setup
      vs workflow-based CodeQL (disable default setup if going workflow-based),
      add the dependency-review config, and fix/replace the go-audit step
      (e.g. `golang.org/x/vuln/cmd/govulncheck`) — ideally upstream in ghcommon.
