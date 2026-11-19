# Relaunching picks up a running task (R-05); it runs out, the alert appears, and Mark
# as completed shows a message. demoseed --running leaves a task with 8 s to go.
Output docs/demo/alert.gif
Source docs/demo/settings.tape

Hide
Type `export PS1='$ ' D=$(mktemp -d) && ./bin/demoseed --data-dir "$D" --running 9 >/dev/null && clear`
Enter
Sleep 1s
Type `PATH=/var/empty ./bin/unitock --data-dir "$D"`
Enter
Wait+Screen /Stretch break/
Show

Wait+Screen@15s /Time.s up/
Sleep 2.5s
Right
Sleep 1s
Enter
Sleep 3s
Type "x"
Sleep 1.5s

Hide
Ctrl+C
