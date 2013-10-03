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
)

var (
	headertpl = `
<!DOCTYPE html>
<html>
<head>
<title>alpha status</title>
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
<h1 id="title">alpha.offblast.org</h1>
<div id="output">
{{ range . }}
<pre># {{.Cmd}}
{{ range .Output }} {{ . }}
{{ end }}
</pre>
{{ end }}
<pre>#^D</pre>
</div>
</div>
`
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

func status(w http.ResponseWriter, r *http.Request) {
	var err error
	var outputs []CommandOutput
	var hdr, stat, footer *template.Template

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in status", r)
		}
	}()

	wr := bufio.NewWriter(w)

	cmds := []string{
		"cat /etc/motd",
		`who | awk '{ printf "%-10s    %s  %s %s %s\n", $1, $2, $3, $4, $5, $6 }'`,
		"uptime",
		"downtimes",
		"df -h",
	}

	if hdr, err = template.New("header").Parse(headertpl); err != nil {
		goto error
	} else if err = hdr.Execute(wr, nil); err != nil {
		goto error
	}

	for _, c := range cmds {
		outputs = append(outputs, RunSh(c))
	}

	if stat, err = template.New("status").Parse(statustpl); err != nil {
		goto error
	} else if err = stat.Execute(wr, outputs); err != nil {
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
