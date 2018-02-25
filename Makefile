# Colors
NOCOLOR=\033[0m
RED=\033[0;31m
GREEN=\033[0;32m

clean:
	@go clean

build:
	@echo "Building project"
	@go build && echo "${GREEN}Success!${NOCOLOR}" || echo "${RED}Build failed!${NOCOLOR}";

debug_in_tmux_pane:
	@echo "Launching delve debugger in bottom-right tmux pane"
	@if pgrep dlv >/dev/null 2>&1; then killall dlv; fi
	@tmux send-keys -t bottom-right 'echo "Magic launching Delve debugger"' Enter
	@tmux send-keys -t bottom-right 'dlv exec portray -- ${PORTRAY_ARGS}' ENTER

exec_in_tmux_pane:
	@echo "Executing in bottom-right tmux pane"
	@if pgrep dlv >/dev/null 2>&1; then killall dlv; fi
	@tmux send-keys -t bottom-right './portray ${PORTRAY_ARGS}' ENTER

execloop:
	@echo "Starting file watcher"
	@fswatch --exclude='.*\.git' \
    --exclude='.*\.yaml' \
    --exclude='.*\.json' \
	--exclude='.*\.swp' \
	--exclude='.*\debug.*?' \
	--exclude='.*4913' \
	--exclude='Makefile' \
	--exclude='LICENSE' \
	--exclude='.*/portray/portray' \
	--recursive . | \
	xargs -n1 -I{} sh -c 'echo "Change detected: {}"; make clean; make build; if [ -f portray ]; then make exec_in_tmux_pane PORTRAY_ARGS="${PORTRAY_ARGS}"; fi'

debugloop:
	@echo "Starting file watcher"
	@fswatch --exclude='.*\.git' \
    --exclude='.*\.yaml' \
    --exclude='.*\.json' \
	--exclude='.*\.swp' \
	--exclude='.*\debug.*?' \
	--exclude='.*4913' \
	--exclude='Makefile' \
	--exclude='LICENSE' \
	--exclude='.*/portray/portray' \
	--recursive . | \
	xargs -n1 -I{} sh -c 'echo "Change detected: {}"; make clean; make build; if [ -f portray ]; then make debug_in_tmux_pane PORTRAY_ARGS="${PORTRAY_ARGS}"; fi'
