# Ubuntu Herd MVP - Quick Start for LLM

## INSTRUCTIONS FOR LLM

You are about to build Ubuntu Herd MVP - a complete working PHP development environment tool.

### BEFORE YOU START

1. Read DEVELOPMENT_ROADMAP.md completely
2. Have COMPLETE_CODE.md open as reference
3. Follow phases in STRICT ORDER
4. Test with `go build` after each phase
5. Do NOT skip any files

---

## PHASE-BY-PHASE CHECKLIST

### Phase 0: Setup (15 min)
- [ ] Create project directory: `mkdir herd && cd herd`
- [ ] Initialize go module: `go mod init github.com/thesmartyking/herd`
- [ ] Run: `go get -u github.com/spf13/cobra@latest`
- [ ] Run: `go get -u github.com/sirupsen/logrus@latest`
- [ ] Run: `go get -u gopkg.in/yaml.v3@latest`
- [ ] Create directories per roadmap
- [ ] Create main.go from COMPLETE_CODE.md
- [ ] Test: `go build -o herd`

### Phase 1: Core Packages (30 min)
- [ ] Copy pkg/php/manager.go
- [ ] Copy pkg/composer/parser.go
- [ ] Copy pkg/nginx/config.go
- [ ] Copy pkg/services/manager.go
- [ ] Create pkg/config/loader.go from COMPLETE_CODE.md
- [ ] Create pkg/config/sites.go from COMPLETE_CODE.md
- [ ] Create pkg/utils/helpers.go from COMPLETE_CODE.md
- [ ] Test: `go build ./pkg/...`

### Phase 2: Commands (30 min)
- [ ] Create cmd/root.go from COMPLETE_CODE.md
- [ ] Create cmd/status.go from COMPLETE_CODE.md
- [ ] Create cmd/logs.go from COMPLETE_CODE.md
- [ ] Create cmd/sites.go from COMPLETE_CODE.md
- [ ] Create cmd/control.go from COMPLETE_CODE.md
- [ ] Test: `go build ./cmd/...`

### Phase 3: Init Command (20 min)
- [ ] Create cmd/init.go from COMPLETE_CODE.md
- [ ] Test: `go build -o herd && ./herd --help`
- [ ] Verify: `./herd init --help`

### Phase 4: Serve Command (20 min)
- [ ] Create cmd/serve.go from COMPLETE_CODE.md
- [ ] Test: `go build -o herd`
- [ ] Verify: `./herd serve --help`

### Phase 5: Integration (10 min)
- [ ] Run `go mod tidy`
- [ ] Run `go build -o herd`
- [ ] Test each command: `./herd --help`
- [ ] Verify no compilation errors

### Phase 6: Testing (15 min)
- [ ] Create test.sh from roadmap
- [ ] Run manual tests
- [ ] Verify error messages make sense
- [ ] Check all commands exist

### Phase 7: Polish (10 min)
- [ ] Update help text as needed
- [ ] Add descriptive errors
- [ ] Create README.md

---

## CRITICAL RULES

✅ DO THIS:
- Follow phases in exact order
- Copy code blocks verbatim
- Test after each phase
- Create directories as shown
- Handle all errors gracefully

❌ DON'T DO THIS:
- Skip phases
- Modify code "to improve it"
- Assume files already exist
- Skip error handling
- Create files in wrong order

---

## Testing After Each Phase

```bash
# Phase 0: Can it compile?
go build -o herd

# Phase 1: Do packages compile?
go build ./pkg/...

# Phase 2: Do commands exist?
./herd --help

# Phase 3: Does init help work?
./herd init --help

# Phase 4: Does serve help work?
./herd serve --help

# Phase 5-7: Full build
go build -o herd && ./herd --help
```

---

## Expected Final Output

When complete, these should work:

```bash
$ ./herd --help
Ubuntu Herd - Local PHP/Laravel development environment
...

$ ./herd init --help
Initialize Herd environment
...

$ ./herd serve --help
Register and serve a project
...

$ ./herd status
📊 Herd Status
PHP Versions:
  ✓ PHP 8.2
Registered Sites:
  (none)

$ ./herd sites
No sites registered yet. Run: herd serve <name> <path>
```

---

## If Something Goes Wrong

1. **Compilation error**: Check import statements match COMPLETE_CODE.md exactly
2. **Missing file**: Verify file is in correct directory
3. **Command not found**: Ensure cmd file is imported in cmd/root.go
4. **Logic error**: Read error message, check DEVELOPMENT_ROADMAP.md for context

---

## Success Criteria

Mark complete when:
- [x] `go build -o herd` succeeds with no errors
- [x] `./herd --help` shows all commands
- [x] `./herd init --help` shows init description
- [x] `./herd serve --help` shows serve description
- [x] `./herd status` runs without errors
- [x] `./herd sites` shows list placeholder
- [x] All .go files are syntactically correct

---

## Timeline

Total time: 6-8 hours of actual coding

- Phase 0: 15 minutes
- Phase 1: 30 minutes
- Phase 2: 30 minutes
- Phase 3: 20 minutes
- Phase 4: 20 minutes
- Phase 5: 10 minutes
- Phase 6: 15 minutes
- Phase 7: 10 minutes
- Phase 8: 1-2 hours (buffer/fixes/testing)

---

## Your Job as LLM

1. Create each file exactly as shown in COMPLETE_CODE.md
2. Don't skip or combine phases
3. Test compilation after each phase
4. Report any errors immediately
5. Follow error messages to fix issues
6. When complete, verify all commands work

---

## START HERE

1. Create project directory
2. Copy the exact commands from Phase 0
3. Create main.go from COMPLETE_CODE.md
4. Run `go build -o herd`
5. If successful, continue to Phase 1

Let's build! 🚀
