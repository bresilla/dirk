//inspired by https://github.com/briandowns/spinner
package dirk

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/fatih/color"
)

var SpinnerSets = map[int][]string{
	0:  {"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"},
	1:  {"▁", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃", "▁"},
	2:  {"▖", "▘", "▝", "▗"},
	3:  {"┤", "┘", "┴", "└", "├", "┌", "┬", "┐"},
	4:  {"◢", "◣", "◤", "◥"},
	5:  {"◰", "◳", "◲", "◱"},
	6:  {"◴", "◷", "◶", "◵"},
	7:  {"◐", "◓", "◑", "◒"},
	8:  {".", "o", "O", "@", "*"},
	9:  {"|", "/", "-", "\\"},
	10: {"◡◡", "⊙⊙", "◠◠"},
	11: {"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"},
	12: {">))'>", " >))'>", "  >))'>", "   >))'>", "    >))'>", "   <'((<", "  <'((<", " <'((<"},
	13: {"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"},
	14: {"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
	15: {"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"},
	16: {"▉", "▊", "▋", "▌", "▍", "▎", "▏", "▎", "▍", "▌", "▋", "▊", "▉"},
	17: {"■", "□", "▪", "▫"},
	18: {"←", "↑", "→", "↓"},
	19: {"╫", "╪"},
	20: {"⇐", "⇖", "⇑", "⇗", "⇒", "⇘", "⇓", "⇙"},
	21: {"⠁", "⠁", "⠉", "⠙", "⠚", "⠒", "⠂", "⠂", "⠒", "⠲", "⠴", "⠤", "⠄", "⠄", "⠤", "⠠", "⠠", "⠤", "⠦", "⠖", "⠒", "⠐", "⠐", "⠒", "⠓", "⠋", "⠉", "⠈", "⠈"},
	22: {"⠈", "⠉", "⠋", "⠓", "⠒", "⠐", "⠐", "⠒", "⠖", "⠦", "⠤", "⠠", "⠠", "⠤", "⠦", "⠖", "⠒", "⠐", "⠐", "⠒", "⠓", "⠋", "⠉", "⠈"},
	23: {"⠁", "⠉", "⠙", "⠚", "⠒", "⠂", "⠂", "⠒", "⠲", "⠴", "⠤", "⠄", "⠄", "⠤", "⠴", "⠲", "⠒", "⠂", "⠂", "⠒", "⠚", "⠙", "⠉", "⠁"},
	24: {"⠋", "⠙", "⠚", "⠒", "⠂", "⠂", "⠒", "⠲", "⠴", "⠦", "⠖", "⠒", "⠐", "⠐", "⠒", "⠓", "⠋"},
	25: {"ｦ", "ｧ", "ｨ", "ｩ", "ｪ", "ｫ", "ｬ", "ｭ", "ｮ", "ｯ", "ｱ", "ｲ", "ｳ", "ｴ", "ｵ", "ｶ", "ｷ", "ｸ", "ｹ", "ｺ", "ｻ", "ｼ", "ｽ", "ｾ", "ｿ", "ﾀ", "ﾁ", "ﾂ", "ﾃ", "ﾄ", "ﾅ", "ﾆ", "ﾇ", "ﾈ", "ﾉ", "ﾊ", "ﾋ", "ﾌ", "ﾍ", "ﾎ", "ﾏ", "ﾐ", "ﾑ", "ﾒ", "ﾓ", "ﾔ", "ﾕ", "ﾖ", "ﾗ", "ﾘ", "ﾙ", "ﾚ", "ﾛ", "ﾜ", "ﾝ"},
	26: {".", "..", "..."},
	27: {"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▉", "▊", "▋", "▌", "▍", "▎", "▏", "▏", "▎", "▍", "▌", "▋", "▊", "▉", "█", "▇", "▆", "▅", "▄", "▃", "▂", "▁"},
	28: {".", "o", "O", "°", "O", "o", "."},
	29: {"+", "x"},
	30: {"v", "<", "^", ">"},
	31: {"🌍", "🌎", "🌏"},
	32: {"◜", "◝", "◞", "◟"},
	33: {"⬒", "⬔", "⬓", "⬕"},
	34: {"⬖", "⬘", "⬗", "⬙"},
	35: {"♠", "♣", "♥", "♦"},
}

// Spinner struct to hold the provided options
type Spinner struct {
	Percent    int
	Delay      time.Duration                 // Delay is the speed of the indicator
	chars      []string                      // chars holds the chosen character set
	Prefix     string                        // Prefix is the text preppended to the indicator
	Suffix     string                        // Suffix is the text appended to the indicator
	lastOutput string                        // last character(set) written
	color      func(a ...interface{}) string // default color is white
	lock       *sync.RWMutex                 //
	Writer     io.Writer                     // to make testing better, exported so users have access
	active     bool                          // active holds the state of the spinner
	stopChan   chan struct{}                 // stopChan is a channel used to stop the indicator
}

// New provides a pointer to an instance of Spinner with the supplied options
func NewSpinner(cs []string, d time.Duration, i int) *Spinner {
	s := &Spinner{
		Delay:    d,
		Percent:  i,
		chars:    cs,
		color:    color.New(color.FgWhite).SprintFunc(),
		lock:     &sync.RWMutex{},
		Writer:   color.Output,
		active:   false,
		stopChan: make(chan struct{}, 1),
	}
	return s
}

// Active will return whether or not the spinner is currently active
func (s *Spinner) Active() bool {
	return s.active
}

// Start will start the indicator
func (s *Spinner) Start() {
	s.lock.Lock()
	if s.active {
		s.lock.Unlock()
		return
	}
	s.active = true
	s.lock.Unlock()

	go func() {
		for {
			for i := 0; i < len(s.chars); i++ {
				select {
				case <-s.stopChan:
					return
				default:
					s.lock.Lock()
					s.erase()
					var outColor string
					if runtime.GOOS == "windows" {
						if s.Writer == os.Stderr {
							outColor = fmt.Sprintf("\r%s%s%s ", s.Prefix, s.chars[i], s.Suffix)
						} else {
							outColor = fmt.Sprintf("\r%s%s%s ", s.Prefix, s.color(s.chars[i]), s.Suffix)
						}
					} else {
						outColor = fmt.Sprintf("%s%s%s ", s.Prefix, s.color(s.chars[i]), s.Suffix)
					}
					outPlain := fmt.Sprintf("%s%s%s ", s.Prefix, s.chars[i], s.Suffix)
					fmt.Fprint(s.Writer, outColor)
					s.lastOutput = outPlain
					delay := s.Delay
					s.lock.Unlock()

					time.Sleep(delay)
				}
			}
		}
	}()
}

// Stop stops the indicator
func (s *Spinner) Stop() {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.active {
		s.active = false
		s.erase()
		s.stopChan <- struct{}{}
	}
}

// Restart will stop and start the indicator
func (s *Spinner) Restart() {
	s.Stop()
	s.Start()
}

// Reverse will reverse the order of the slice assigned to the indicator
func (s *Spinner) Reverse() {
	s.lock.Lock()
	defer s.lock.Unlock()
	for i, j := 0, len(s.chars)-1; i < j; i, j = i+1, j-1 {
		s.chars[i], s.chars[j] = s.chars[j], s.chars[i]
	}
}

// Color will set the struct field for the given color to be used
func (s *Spinner) Color(colors ...string) error {
	colorAttributes := make([]color.Attribute, len(colors))
	s.color = color.New(colorAttributes...).SprintFunc()
	s.Restart()
	return nil
}

// UpdateSpeed will set the indicator delay to the given value
func (s *Spinner) UpdateSpeed(d time.Duration) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.Delay = d
}

// UpdateCharSet will change the current character set to the given one
func (s *Spinner) UpdateCharSet(cs []string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.chars = cs
}

// erase deletes written characters
func (s *Spinner) erase() {
	n := utf8.RuneCountInString(s.lastOutput)
	if runtime.GOOS == "windows" {
		clearString := "\r"
		for i := 0; i < n; i++ {
			clearString += " "
		}
		fmt.Fprintf(s.Writer, clearString)
		return
	}
	del, _ := hex.DecodeString("7f")
	for _, c := range []string{
		"\b",
		string(del),
		"\b",
		"\033[K", // for macOS Terminal
	} {
		for i := 0; i < n; i++ {
			fmt.Fprintf(s.Writer, c)
		}
	}
	s.lastOutput = ""
}
