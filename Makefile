default:
	./task

# Delegates every make target to the equivalent ./task target.
# Intentional semantic changes from the old Makefile:
#   make fmt  → ./task fmt  (full format, was incremental; use make fmt-q for incremental)
#   make lint → ./task lint (full lint,   was incremental; use make lint-q for incremental)
.DEFAULT:
	@./task "$@"
