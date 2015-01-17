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
000 (github.com/ericchiang/pup.AttrDisplayer).Display
000 (github.com/ericchiang/pup.TreeDisplayer).Display
000 github.com/ericchiang/pup.init
000 github.com/ericchiang/pup.main
001 (github.com/ericchiang/pup.JSONDisplayer).Display
001 (github.com/ericchiang/pup.TextDisplayer).Display
...
028 github.com/ericchiang/pup.ParseClassMatcher
028 github.com/ericchiang/pup.ParseIdMatcher
028 github.com/ericchiang/pup.ParsePseudo
033 strconv.Atoi
051 (*regexp.Regexp).FindAllStringSubmatch
054 (*bytes.Buffer).WriteRune
086 regexp.MustCompile
088 fmt.Errorf
089 regexp.QuoteMeta
201 strings.IndexRune
205 (*bytes.Buffer).String
265 (*text/scanner.Scanner).Next
```
