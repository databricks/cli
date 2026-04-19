TASK := go tool -modfile=tools/task/go.mod task
.PHONY: $(MAKECMDGOALS)
%:
	@$(TASK) "$@"
