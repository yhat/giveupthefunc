#giveupthefunc

Go function use profiling.

How many times are you really using that?

##Usage

[giveupthefunc](https://www.youtube.com/watch?v=jJvjWh2Vhu4)
determines the number of times a function is called with some set of packages.

You specify the scope of your analysis using a regexp with the `-p` flag.
For each package and dependency this flag matches, giveupthefunc will
track all function declarations and all function uses within that package.
It then prints a summary of those stats for all the packages.

Here's an example using one of my own packages, setting the scope to my
github repos.

```
$ giveupthefunc -p='github.com/ericchiang' github.com/ericchiang/pup
ANALYZING github.com/ericchiang/pup
USAGE:
000 github.com/ericchiang/pup.init
000 github.com/ericchiang/pup.isVoidElement
000 github.com/ericchiang/pup.main
001 bytes.init
001 encoding/json.init
001 fmt.Println
001 fmt.init
001 github.com/ericchiang/pup.PrintHelp
001 github.com/ericchiang/pup.SelectFromChildren
001 github.com/ericchiang/pup.SelectNextSibling
001 github.com/ericchiang/pup.init#1
001 github.com/fatih/color.init
...
051 (*regexp.Regexp).FindAllStringSubmatch
054 (*bytes.Buffer).WriteRune
070 regexp.MustCompile
082 fmt.Errorf
137 (*bytes.Buffer).String
201 strings.IndexRune
254 (*text/scanner.Scanner).Next
```
