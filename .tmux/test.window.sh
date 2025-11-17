# Set window root path. Default is `$session_root`.
# Must be called before `new_window`.
window_root "$PWD"

# Create new window. If no argument is given, window name will be based on
# layout file name.
new_window "test"

# Split window into panes.
split_v 20

# Run commands.
#run_cmd "top"     # runs in active pane
#run_cmd  "~/code/zed/target/release/Zed" #"nvim ." 1 # runs in pane 1
run_cmd "sleep 5" 1
run_cmd "just test-infra" 1

# Paste text
#send_keys "top"    # paste into active pane
#send_keys "date" 1 # paste into pane 1
send_keys "cat .tmp/core.log" 2

# Set active pane.
select_pane 1
