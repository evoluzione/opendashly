#!/bin/sh
set -u

# Run the Go API (:8080) and the SvelteKit node server (:5173) side by side and
# tear the WHOLE container down as soon as EITHER one exits.
#
# Previously the backend ran in the background while `node build` was PID 1: a
# backend crash (e.g. a runtime panic) left PID 1 alive, so the container stayed
# "Up", port 8080 went dead with no listener, and `restart: unless-stopped`
# never fired (it only triggers on PID 1 exit). The dashboard would load from
# :5173 but every API call got connection-refused until a manual restart.
#
# Supervising both processes and exiting on the first death lets Docker's restart
# policy recover the container automatically.

/app/backend &
backend_pid=$!

cd /app/frontend
node build &
frontend_pid=$!

terminate() {
	kill "$backend_pid" "$frontend_pid" 2>/dev/null || true
}
trap terminate TERM INT

# Poll instead of `wait -n`, which busybox ash (this image's /bin/sh) lacks.
while kill -0 "$backend_pid" 2>/dev/null && kill -0 "$frontend_pid" 2>/dev/null; do
	sleep 2
done

if ! kill -0 "$backend_pid" 2>/dev/null; then
	echo "entrypoint: backend (:8080) exited; shutting down container for restart" >&2
else
	echo "entrypoint: frontend (:5173) exited; shutting down container for restart" >&2
fi

terminate
wait
exit 1
