Output docs/tui.gif

Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 14
Set LineHeight 1.2
Set Width 960
Set Height 580
Set Padding 20
Set WindowBar Colorful
Set Framerate 24
Set Theme {"name":"tang","black":"#171717","red":"#e06c75","green":"#98c379","yellow":"#e5c07b","blue":"#61afef","purple":"#c678dd","cyan":"#56b6c2","white":"#abb2bf","brightBlack":"#636d83","brightRed":"#ff7b86","brightGreen":"#b5e890","brightYellow":"#ffd68a","brightBlue":"#7ecbff","brightPurple":"#d896ff","brightCyan":"#70d6e3","brightWhite":"#ffffff","background":"#171717","foreground":"#d8dee9","selection":"#3e4451","cursor":"#abb2bf"}

Hide
Type "export PS1='$ '"
Enter
Type "tang() { ./bin/tang -f sample/artifacts/gotest_json.out -replay -rate 0.5 -slow-threshold 3s; }"
Enter
Type "clear"
Enter
Sleep 200ms
Show

Sleep 300ms
Type "tang test ./..."
Sleep 300ms
Enter
Sleep 8.2s
