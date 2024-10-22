# Set window root path. Default is `$session_root`.
# Must be called before `new_window`.
window_root "~/.config/mir"

# Create new window. If no argument is given, window name will be based on
# layout file name.
new_window "cfg"

# Split window into panes.
split_v 20
#split_h 50

# Run commands.
#run_cmd "top"     # runs in active pane
#run_cmd  "~/code/zed/target/release/Zed" #"nvim ." 1 # runs in pane 1

# Paste text
#send_keys "top"    # paste into active pane
#send_keys "date" 1 # paste into pane 1
#send_keys "nvim ." 1

# Set active pane.
select_pane 1
run_cmd "nvim ."
