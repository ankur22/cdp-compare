# CDP-Compare

The logs needs to be normalised as they are in the example *.log files that are in this project.

## Regex

For k6 logs, perform these steps in VSCode:

1. Find `(.*(<- |-> ).*)`; Select all the lines; cut them from the file; Delete all remaining lines; paste the lines that were cut.
2. Find `(time=.*)(-> .*)` and replace with `$2`
3. Find `(time=.*)(<- .*)` and replace with `$2`
4. Find `" category=".*` and replace with an empty value (i.e. remove).
5. Find `\\"` and replace with `"`.
6. Find `\\\\"` and replace with an empty value (i.e. remove).

## Run

`go run main.go <filenameA> <filenameB>`

e.g. `go run main.go newtab2-cdp-k6.log newtab2-cdp-pw.log`
