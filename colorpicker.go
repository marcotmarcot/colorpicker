package main

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func main() {
	rand.Seed(time.Now().Unix())
	http.HandleFunc("/", handler)
	http.HandleFunc("/wait", waitHandler)
	log.Fatal(http.ListenAndServe(":1234", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	round, err := getRound(r)
	if err != nil {
		fmt.Fprintf(w, "%v", err)
		return
	}
	id, err := getId(r, round)
	if err != nil {
		fmt.Fprintf(w, "%v", err)
		return
	}
	s, err := processColor(r.FormValue("color"), id, round - 1)
	if err != nil {
		fmt.Fprintf(w, "%v", err)
		return
	}
	switch s {
	case lose:
		printLose(w, r)
	case wait:
		http.Redirect(w, r, fmt.Sprintf("/wait?id=%s&round=%d", id, round), 307)
	case win:
		printWin(w, r, round, id)
	default:
		fmt.Fprintf(w, "invalid status.")
	}
}

func printLose(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("lose.html")
	if err != nil {
		fmt.Fprintf(w, "%v", err)
		return
	}
	t.Execute(w, struct {
		Link string
	}{"http://" + r.Host})
}

func printWin(w http.ResponseWriter, r *http.Request, round int, id string) {
	c := numberColor(round)
	t, err := template.ParseFiles("index.html")
	if err != nil {
		fmt.Fprintf(w, "%v", err)
		return
	}
	t.Execute(w, struct {
		Round       int
		Link, Color, Id string
	}{round + 1, getLink(r, id, round), c, id})
}

func getRound(r *http.Request) (int, error) {
	str := r.FormValue("round")
	if str == "" {
		return 0, nil
	}
	round, err := strconv.Atoi(str)
	if err != nil {
		return 0, err
	}
	return round, nil
}

func getId(r *http.Request, round int) (string, error) {
	f := r.FormValue("id")
	if f == "" {
		if round == 0 {
			id := strconv.Itoa(rand.Int())
			games[id] = &game{}
			return id, nil
		}
		return "", errors.New("id is required.")
	}
	return f, nil
}

func processColor(color, id string, round int) (status, error) {
	if color == "" {
		return win, nil
	}
	if round == -1 {
		return lose, errors.New("color should not be set in first round.")
	}
	game := games[id]
	if game.round != round {
		return lose, errors.New("invalid round")
	}
	if game.color == "" {
		game.color = color
		return wait, nil
	}
	if strings.ToLower(color) == strings.ToLower(game.color) {
		game.round++
		game.color = ""
		return win, nil
	}
	game.round = -1
	return lose, nil
}

type status int

const (
	win = iota
	lose
	wait
)


func getLink(r *http.Request, id string, round int) string {
	if round != 0 {
		return ""
	}
	if r.FormValue("id") != "" {
		return ""
	}
	return "http://" + r.Host + r.RequestURI + "?id=" + id
}

func numberColor(n int) string {
	return modifierColor(numberModifier(n))
}

func numberModifier(n int) modifier {
	i := 0
	if i >= n {
		return triple(0)
	}
	for reference := 1; true; reference++ {
		i++
		if i >= n {
			return triple(reference)
		}
		for left := 0; left < reference; left++ {
			for right := 0; right < reference; right++ {
				for pos := 0; pos < 3; pos++ {
					i++
					if i >= n {
						return single(reference, left, right, pos)
					}
				}
			}
		}
		for other := 0; other < reference; other++ {
			for pos := 0; pos < 3; pos++ {
				i++
				if i >= n {
					return double(reference, other, pos)
				}
			}
		}
	}
	log.Fatal("Should never happen")
	return modifier{}
}

type modifier struct {
	red   int
	green int
	blue  int
}

func single(reference, left, right, pos int) modifier {
	switch pos {
	case 0:
		return modifier{reference, left, right}
	case 1:
		return modifier{left, reference, right}
	case 2:
		return modifier{left, right, reference}
	}
	log.Fatal("Invalid pos")
	return modifier{}
}

func double(reference, other, pos int) modifier {
	switch pos {
	case 0:
		return modifier{other, reference, reference}
	case 1:
		return modifier{reference, other, reference}
	case 2:
		return modifier{reference, reference, other}
	}
	log.Fatal("Invalid pos")
	return modifier{}
}

func triple(reference int) modifier {
	return modifier{reference, reference, reference}
}

func modifierColor(m modifier) string {
	return "#" + modifierPrimary(m.red) + modifierPrimary(m.green) + modifierPrimary(m.blue)
}

func modifierPrimary(m int) string {
	return fmt.Sprintf("%02x", pairToInt(modifierToPair(m)))
}

func pairToInt(p primaryPair) int {
	return int(255.0 * p.dividend / p.divisor)
}

type primaryPair struct {
	dividend, divisor int
}

func modifierToPair(m int) primaryPair {
	p := primaryPair{0, 1}
	for i := 0; i < m; i++ {
		if p.divisor == 1 {
			p.dividend++
		} else {
			p.dividend += 2
		}
		if p.dividend > p.divisor {
			p.dividend = 1
			p.divisor *= 2
		}
	}
	return p
}

var games = map[string]*game{}

type game struct {
	round int
	color string
}

func waitHandler(w http.ResponseWriter, r *http.Request) {
	round, err := getRound(r)
	if err != nil {
		fmt.Fprintf(w, "%v", err)
		return
	}
	id, err := getId(r, round)
	if err != nil {
		fmt.Fprintf(w, "%v", err)
		return
	}
	log.Print(id)
	log.Print(games[id])
	log.Print(games[id].round)
	switch games[id].round {
	case -1:
		printLose(w, r)
	case round:
		http.Redirect(w, r, fmt.Sprintf("/?id=%s&round=%d", id, round), 307)
	case round - 1:
		printWait(w)
	default:
		fmt.Fprintf(w, "invalid round")
	}
}

func printWait(w http.ResponseWriter) {
	t, err := template.ParseFiles("wait.html")
	if err != nil {
		fmt.Fprintf(w, "%v", err)
		return
	}
	t.Execute(w, nil)
}
