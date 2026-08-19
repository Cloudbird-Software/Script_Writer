.PHONY: setup fmt lint arch test build check typecheck spec-guard \
        db-rebuild db-export golden eval-l1 judge-cal mine optimize dash

setup:      ; uv sync --all-extras
fmt:        ; uv run ruff format . && uv run ruff check --fix .
lint:       ; uv run ruff format --check . && uv run ruff check . && uv run pyright
typecheck:  ; uv run pyright

# 资产层完整性守卫 = 本仓的架构边界 lint（对应组织模板 depcruise 位，见 docs/ARCHITECTURE.md）
spec-guard:
	uv run python -m nsc.guards.spec_reduction
	uv run python -m nsc.guards.checks_schema
	uv run python -m nsc.guards.prompts_untouched
	uv run python -m nsc.guards.ir_schema_diff
	uv run python -m nsc.guards.budgets
	uv run python -m nsc.guards.rules_conflict
	uv run python -m nsc.guards.db_export_fresh

arch: spec-guard
test:       ; $(MAKE) --no-print-directory db-rebuild && uv run pytest -m "not llm" -n auto --cov
build:      ; uv build
check:      lint arch test

db-rebuild: ; uv run nsc db rebuild
db-export:  ; uv run nsc db export
golden:     ; $(MAKE) --no-print-directory db-rebuild && uv run pytest -m golden -n auto
eval-l1:    ; uv run nsc eval l1 --sample $${SAMPLE:-12}
judge-cal:  ; uv run nsc judge calibrate --report out/judge_calibration.md
mine:       ; uv run nsc mine run --open-pr
optimize:   ; uv run nsc optimize --pass $${PASS:-p5_dialogue} --auto light
dash:       ; uv run nsc metrics weekly --write docs/metrics/
