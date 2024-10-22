# Set window root path. Default is `$session_root`.
# Must be called before `new_window`.
window_root "$PWD/infra/surrealdb/"

# Create new window. If no argument is given, window name will be based on
# layout file name.
new_window "surrealdb"

run_cmd "docker compose up" # runs in active pane

split_v 20

send_keys ""

# Paste text
#send_keys "top"    # paste into active pane
#send_keys "date" 1 # paste into pane 1

# Set active pane.
#select_pane 0
