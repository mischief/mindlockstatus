package main

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"os/exec"
	"strings"
	"math/rand"
)

var (
	headertpl = `
<!DOCTYPE html>
<html>
<head>
<title>{{ .Hostname }}</title>
<style>
body {
	color: #00FF00;
	background: #000000;
	font-family: Sans-Serif;
}

a:link, a:hover, a:active {
	color: green;
}

a:visited {
	color: #00FF00;
}

#title {
	margin: 40px auto;
}

#center {
	width: -moz-min-content; width: -webkit-min-content; width: min-content;
	margin: 40px auto;
}
</style>
<script>
/* make some links */
function urlize(elem) {
	var upat = /((http|ftp|https):\/\/[\w-]+(\.[\w-]+)+([\w.,@?^=%&amp;:\/~+#-]*[\w@?^=%&amp;\/~+#-])?)/g;
	var e = document.getElementById(elem);
	e.innerHTML = e.innerHTML.replace(upat, '<a href="$1">$1</a>');
}
</script>
</head>
<body onload="urlize('center');">
`
	footertpl = `
</body>
</html>
`

	statustpl = `
<div id="center">
<div id="title">
	<h1>{{.Hostname}}</h1>
	<h2>{{.Quote}}</h2>
</div>
<div id="output">
{{ range .Outputs }}
<pre># {{.Cmd}}
{{ range .Output }} {{ . }}
{{ end }}
</pre>
{{ end }}
<pre>#^D</pre>
</div>
</div>
`

	cmds = []string{
		"cat /etc/motd",
		`who | awk '{ printf "%-10s    %s  %s %2s %s\n", $1, $2, $3, $4, $5 }'`,
		"uptime",
		"downtimes",
		"df -h",
	}

	quotes = []string{
		"i need something stronger.",
		"for more enjoyment and greater efficiency, consumption is being standardized. we are sorry ...",
		"everything will be all right. you are in my hands. i am here to protect you. you have nowhere to go. you have nowhere to go.",
		"you have nowhere to go. i am here to protect you.",
		"how shall the new environment be programmed? it all happened so slowly that most men failed to realize that anything had happened at all.",
		"take four red capsules. in 10 minutes take two more. help is on the way.",
		"control center 626 holds no responsibility... for error in mindlock.",
	}
)

type CommandOutput struct {
	Cmd    string
	Output []string
}

func RunSh(cmd string) CommandOutput {
	c := exec.Command("/bin/sh", "-c", cmd)

	output, err := c.CombinedOutput()
	if err != nil {
		return CommandOutput{cmd, []string{err.Error()}}
	}

	buf := bytes.NewBuffer(output)

	sc := bufio.NewScanner(buf)

	var out []string
	for sc.Scan() {
		out = append(out, strings.TrimSpace(sc.Text()))
	}

	return CommandOutput{cmd, out}
}

type Header struct {
	Hostname string
}

type Status struct {
	Hostname string
	Quote string
	Outputs []CommandOutput
}

func status(w http.ResponseWriter, r *http.Request) {
	var err error
	var headdat Header
	var statdat Status
	var outputs []CommandOutput
	var hdr, stat, footer *template.Template

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in status", r)
		}
	}()

	wr := bufio.NewWriter(w)

	statdat.Hostname = r.Host
	if statdat.Hostname == "" {
		statdat.Hostname, _ = os.Hostname() 
	}

	headdat.Hostname = statdat.Hostname

	statdat.Quote = quotes[rand.Intn(len(quotes))]

	if hdr, err = template.New("header").Parse(headertpl); err != nil {
		goto error
	} else if err = hdr.Execute(wr, headdat); err != nil {
		goto error
	}

	for _, c := range cmds {
		outputs = append(outputs, RunSh(c))
	}

	statdat.Outputs = outputs

	if stat, err = template.New("status").Parse(statustpl); err != nil {
		goto error
	} else if err = stat.Execute(wr, statdat); err != nil {
		goto error
	}

	if footer, err = template.New("footer").Parse(footertpl); err != nil {
		goto error
	} else if err = footer.Execute(wr, nil); err != nil {
		goto error
	}

error:
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	wr.Flush()

}

func main() {
	listener, _ := net.Listen("tcp", "127.0.0.1:9001")
	http.HandleFunc("/", status)
	fcgi.Serve(listener, nil)
}
