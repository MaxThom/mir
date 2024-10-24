# Set window root path. Default is `$session_root`.
# Must be called before `new_window`.
window_root "$PWD/book"

# Create new window. If no argument is given, window name will be based on
# layout file name.
new_window "book"

# Split window into panes.
split_v 80
#split_h 50

# Run commands.
#run_cmd "top"     # runs in active pane
#run_cmd  "~/code/zed/target/release/Zed" #"nvim ." 1 # runs in pane 1
run_cmd "mdbook serve -p 5001" 1
run_cmd "git status"

# Paste text
#send_keys "top"    # paste into active pane
#send_keys "date" 1 # paste into pane 1
#send_keys "nvim ." 1

# Set active pane.
#select_pane 1
