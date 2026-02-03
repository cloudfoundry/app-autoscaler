#!/usr/bin/env bash
# shellcheck disable=SC2154
#
# Run MTA acceptance tests in parallel via CF tasks
# Usage: make mta-acceptance-tests
#        SUITES="api app" NODES=8 GINKGO_OPTS="--fail-fast" make mta-acceptance-tests

set -euo pipefail

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
# shellcheck source=scripts/vars.source.sh
source "${script_dir}/vars.source.sh"
# shellcheck source=scripts/common.sh
source "${script_dir}/common.sh"

# Configuration
SUITES="${SUITES:-api app broker}"
NODES="${NODES:-4}"
GINKGO_OPTS="${GINKGO_OPTS:-}"
SKIP_TEARDOWN="${SKIP_TEARDOWN:-false}"
POLL_INTERVAL=60
MAX_WAIT_TIME=3600

# Task tracking
script_start_time=$(date +%s)
APP_GUID=""
LAUNCHED_TASK_GUIDS=()

validate() {
	step "Validating prerequisites"
	command -v cf &>/dev/null || { echo "ERROR: cf CLI not found"; exit 1; }
	command -v jq &>/dev/null || { echo "ERROR: jq not found"; exit 1; }

	echo "  ✓ Prerequisites validated"
}

validate_app() {
	step "Validating acceptance-tests app"
	# Get app GUID and validate app exists
	APP_GUID=$(cf app acceptance-tests --guid 2>/dev/null) || { echo "ERROR: acceptance-tests app not found. Run: make mta-deploy"; exit 1; }
	[[ -n "${APP_GUID}" ]] || { echo "ERROR: acceptance-tests app not found. Run: make mta-deploy"; exit 1; }

	echo "  ✓ App found (GUID: ${APP_GUID:0:8})"
}

launch_task() {
	local suite=$1
	local task_name
	task_name="run-acceptance-${suite}-pr${PR_NUMBER:-unknown}-$(date +%s)-${RANDOM}"
	local cmd
	cmd="SUITES='${suite}' NODES=${NODES} GINKGO_OPTS='${GINKGO_OPTS}' SKIP_TEARDOWN=${SKIP_TEARDOWN} bash /home/vcap/app/scripts/run-acceptance-tests-task.sh"

	local guid
	local cf_output
	cf_output=$(cf curl "/v3/apps/${APP_GUID}/tasks" -X POST -d "$(jq -n --arg n "${task_name}" --arg c "${cmd}" \
		'{name:$n,command:$c,memory_in_mb:2048,disk_in_mb:2048}')" 2>&1)
	guid=$(echo "$cf_output" | jq -r '.guid // empty')
	if [[ -n "${guid}" ]]; then
		echo "${guid}"
	else
		echo "ERROR: Failed to launch task for ${suite}" >&2
		echo "FAILED"
	fi
}

get_pr_tasks() {
	[[ ${#LAUNCHED_TASK_GUIDS[@]} -eq 0 ]] && return 0

	# Build comma-delimited list of GUIDs for CF API query
	local guid_list
	IFS=,
	guid_list="${LAUNCHED_TASK_GUIDS[*]}"

	local tasks_json
	if ! cf curl "/v3/tasks?guids=${guid_list}" >/dev/null 2>&1; then
		return 0
	fi

	tasks_json=$(cf curl "/v3/tasks?guids=${guid_list}" 2>/dev/null)
	echo "$tasks_json" | jq -r '.resources[]? | "\(.name):\(.state)"' 2>/dev/null || true
}

has_unfinished_tasks() {
	local tasks
	tasks=$(get_pr_tasks)
	echo "$tasks" | grep -qE ":(RUNNING|PENDING)$"
}

has_failed_tasks() {
	local tasks
	tasks=$(get_pr_tasks)
	[[ -n "$tasks" ]] && echo "$tasks" | grep -qvE ":SUCCEEDED$"
}

format_time() { printf "%dm%02ds" $(($1/60)) $(($1%60)); }

show_status() {
	local poll=$1
	local elapsed=$(($(date +%s) - script_start_time))

	local tasks
	tasks=$(get_pr_tasks)

	# Build compact status line
	local running=0 pending=0 succeeded=0 failed=0
	while IFS=: read -r name state; do
		[[ -z "$name" ]] && continue
		case "$state" in
			RUNNING) running=$((running + 1)) ;;
			PENDING) pending=$((pending + 1)) ;;
			SUCCEEDED) succeeded=$((succeeded + 1)) ;;
			FAILED) failed=$((failed + 1)) ;;
		esac
	done <<< "$tasks"

	printf "Poll #%d | Elapsed: %s | Running: %d | Pending: %d | Succeeded: %d | Failed: %d\n" \
		"$poll" "$(format_time ${elapsed})" "$running" "$pending" "$succeeded" "$failed"
}

show_final() {
	local passed=0 failed=0
	echo -e "\n======= FINAL RESULTS ======="
	echo "PR_NUMBER=${PR_NUMBER:-unknown}"

	local tasks
	tasks=$(get_pr_tasks)

	if [[ -z "$tasks" ]]; then
		echo "ERROR: No tasks found for PR ${PR_NUMBER:-unknown}"
		return 1
	fi

	local failed_tasks=()
	while IFS=: read -r name state; do
		[[ -z "$name" ]] && continue
		if [[ "${state}" == "SUCCEEDED" ]]; then
			passed=$((passed + 1))
			printf "✓ %s\n" "${name}"
		else
			failed=$((failed + 1))
			printf "✗ %s: %s\n" "${name}" "${state}"
			failed_tasks+=("${name}")
		fi
	done <<< "$tasks"

	echo "============================="
	local total=$((passed + failed))
	echo "Result: ${passed}/${total} passed"
	[[ ${failed} -eq 0 ]] && echo "✓ All tests passed!" || echo "✗ ${failed} failed!"

	# Show logs for failed tasks
	if [[ ${#failed_tasks[@]} -gt 0 ]]; then
		echo ""
		echo "======= LOGS FOR FAILED TASKS ======="
		for task_name in "${failed_tasks[@]}"; do
			echo ""
			echo "--- Logs for: ${task_name} ---"
			cf logs acceptance-tests --recent 2>/dev/null | grep "${task_name}" || echo "No logs available for ${task_name}"
			echo "--- End logs for: ${task_name} ---"
		done
	fi
}

main() {
	echo "PR_NUMBER=${PR_NUMBER:-unknown}"
	step "Running MTA acceptance tests: ${SUITES}"
	validate
	bbl_login
	cf_login
	cf_target "${autoscaler_org}" "${autoscaler_space}"
	validate_app

	# Launch tasks
	step "Launching CF tasks"
	local launched=0
	for suite in $SUITES; do
		local guid
		guid=$(launch_task "${suite}")
		if [[ "${guid}" != "FAILED" ]]; then
			echo "  ✓ ${suite}: ${guid:0:8}"
			LAUNCHED_TASK_GUIDS+=("${guid}")
			launched=$((launched + 1))
		else
			echo "  ✗ ${suite}: Failed to launch"
		fi
	done

	[[ ${launched} -eq 0 ]] && { echo "ERROR: No tasks launched successfully"; exit 1; }

	# Wait for CF API to index the tasks with retry logic
	echo "Waiting for tasks to be visible in CF API..."
	local task_count=0
	local retry=0
	local max_retries=6  # 6 retries * 5 seconds = 30 seconds max wait

	while [[ ${task_count} -eq 0 && ${retry} -lt ${max_retries} ]]; do
		sleep 5
		task_count=$(get_pr_tasks | wc -l)
		retry=$((retry + 1))
		if [[ ${task_count} -eq 0 ]]; then
			echo "  Retry $retry/$max_retries: Tasks not yet visible, waiting..."
		fi
	done

	echo "Verified ${task_count} of ${#LAUNCHED_TASK_GUIDS[@]} tasks visible in CF API"

	if [[ ${task_count} -eq 0 ]]; then
		echo "ERROR: None of the launched tasks are visible in CF API after ${max_retries} retries"
		exit 1
	fi

	# Poll until all tasks are finished
	step "Monitoring tasks"
	local poll=0
	while has_unfinished_tasks; do
		poll=$((poll + 1))
		[[ $(($(date +%s) - script_start_time)) -gt ${MAX_WAIT_TIME} ]] && {
			echo "ERROR: Timeout after ${MAX_WAIT_TIME}s"
			break
		}
		show_status ${poll}
		sleep ${POLL_INTERVAL}
	done

	# Final summary
	show_final

	# Exit with error if any tasks failed
	if has_failed_tasks; then
		echo "✗ Some tasks failed"
		exit 1
	fi

	echo "✓ All tasks succeeded"
	exit 0
}

main "$@"