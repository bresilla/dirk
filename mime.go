//based on https://github.com/gabriel-vasile/mimetype

package dirk

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/gabriel-vasile/mimetype/matchers/json"
)

// Detect returns the mime type and extension of the provided byte slice.
func Detect(in []byte) (mime, extension string) {
	n := Root.match(in, Root)
	return n.Mime(), n.Extension()
}

// DetectReader returns the mime type and extension of the byte slice read
// from the provided reader.
func DetectReader(r io.Reader) (mime, extension string, err error) {
	in := make([]byte, ReadLimit)
	n, err := r.Read(in)
	if err != nil && err != io.EOF {
		return Root.Mime(), Root.Extension(), err
	}
	in = in[:n]

	mime, ext := Detect(in)
	return mime, ext, nil
}

// DetectFile returns the mime type and extension of the provided file.
func DetectFile(file string) (mime, extension string, err error) {
	f, err := os.Open(file)
	if err != nil {
		return Root.Mime(), Root.Extension(), err
	}
	defer f.Close()

	return DetectReader(f)
}

type (
	// Node represents a node in the matchers tree structure.
	// It holds the mime type, the extension and the function to check whether
	// a byte slice has the mime type
	Node struct {
		mime       string
		extension  string
		matchFunc  matchFunc
		exhaustive bool
		children   []*Node
	}
	matchFunc func([]byte) bool
)

// NewNode creates a new Node
func NewNode(mime, extension string, matchFunc matchFunc, children ...*Node) *Node {
	return &Node{
		mime:      mime,
		extension: extension,
		matchFunc: matchFunc,
		children:  children,
	}
}

// Mime returns the mime type associated with the node
func (n *Node) Mime() string { return n.mime }

// Extension returns the file extension associated with the node
func (n *Node) Extension() string { return n.extension }

// Append adds a new node to the matchers tree
// When a node's matching function passes the check, the node's children are
// also checked in order to find a more accurate mime type for the input
func (n *Node) Append(cs ...*Node) { n.children = append(n.children, cs...) }

// match does a depth-first search on the matchers tree
// it returns the deepest successful matcher for which all the children fail
func (n *Node) match(in []byte, deepestMatch *Node) *Node {
	for _, c := range n.children {
		if c.matchFunc(in) {
			return c.match(in, c)
		}
	}
	return deepestMatch
}

// Tree returns a string representation of the matchers tree
func (n *Node) Tree() string {
	var printTree func(*Node, int) string
	printTree = func(n *Node, level int) string {
		offset := ""
		i := 0
		for i < level {
			offset += "|\t"
			i++
		}
		if len(n.children) > 0 {
			offset += "+"
		}
		out := fmt.Sprintf("%s%s \n", offset, n.Mime())
		for _, c := range n.children {
			out += printTree(c, level+1)
		}

		return out
	}

	return printTree(n, 0)
}

var Root = NewNode("application/octet-stream", "", match_True,
	SevenZ, Zip, Pdf, Ps, Psd, Ogg,
	Png, Jpg, Gif, Webp, Tiff, Bmp, Ico,
	Mp3, Flac, Midi, Ape, MusePack, Amr, Wav, Aiff, Au,
	Mpeg, QuickTime, Mp4, WebM, ThreeGP, Avi, Flv, Mkv,
	Txt, Gzip,
)

// The list of nodes appended to the Root node
var (
	Gzip   = NewNode("application/gzip", "gz", match_Gzip)
	SevenZ = NewNode("application/x-7z-compressed", "7z", match_SevenZ)
	Zip    = NewNode("application/zip", "zip", match_Zip, Xlsx, Docx, Pptx, Epub, Jar)
	Pdf    = NewNode("application/pdf", "pdf", match_Pdf)
	Xlsx   = NewNode("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx", match_Xlsx)
	Docx   = NewNode("application/vnd.openxmlformats-officedocument.wordprocessingml.document", "docx", match_Docx)
	Pptx   = NewNode("application/vnd.openxmlformats-officedocument.presentationml.presentation", "pptx", match_Pptx)
	Epub   = NewNode("application/epub+zip", "epub", match_Epub)
	Jar    = NewNode("application/jar", "jar", match_Jar, Apk)
	Apk    = NewNode("application/vnd.android.package-archive", "apk", match_False)
	Ps     = NewNode("application/postscript", "ps", match_Ps)
	Psd    = NewNode("application/x-photoshop", "psd", match_Psd)
	Ogg    = NewNode("application/ogg", "ogg", match_Ogg)

	Txt  = NewNode("text/plain", "txt", match_Txt, Html, Xml, Php, Js, Lua, Perl, Python, Json, Rtf)
	Xml  = NewNode("text/xml; charset=utf-8", "xml", match_Xml, Svg, X3d, Kml, Collada, Gml, Gpx)
	Json = NewNode("application/json", "json", match_Json)
	Html = NewNode("text/html; charset=utf-8", "html", match_Html)
	Php  = NewNode("text/x-php; charset=utf-8", "php", match_Php)
	Rtf  = NewNode("text/rtf", "rtf", match_Rtf)

	Js     = NewNode("application/javascript", "js", match_Js)
	Lua    = NewNode("text/x-lua", "lua", match_Lua)
	Perl   = NewNode("text/x-perl", "pl", match_Perl)
	Python = NewNode("application/x-python", "py", match_Python)

	Svg     = NewNode("image/svg+xml", "svg", match_False)
	X3d     = NewNode("model/x3d+xml", "x3d", match_False)
	Kml     = NewNode("application/vnd.google-earth.kml+xml", "kml", match_False)
	Collada = NewNode("model/vnd.collada+xml", "dae", match_False)
	Gml     = NewNode("application/gml+xml", "gml", match_False)
	Gpx     = NewNode("application/gpx+xml", "gpx", match_False)

	Png  = NewNode("image/png", "png", match_Png)
	Jpg  = NewNode("image/jpeg", "jpg", match_Jpg)
	Gif  = NewNode("image/gif", "gif", match_Gif)
	Webp = NewNode("image/webp", "webp", match_Webp)
	Tiff = NewNode("image/tiff", "tiff", match_Tiff)
	Bmp  = NewNode("image/bmp", "bmp", match_Bmp)
	Ico  = NewNode("image/x-icon", "ico", match_Ico)

	Mp3      = NewNode("audio/mpeg", "mp3", match_Mp3)
	Flac     = NewNode("audio/flac", "flac", match_Flac)
	Midi     = NewNode("audio/midi", "midi", match_Midi)
	Ape      = NewNode("audio/ape", "ape", match_Ape)
	MusePack = NewNode("audio/musepack", "mpc", match_MusePack)
	Wav      = NewNode("audio/wav", "wav", match_Wav)
	Aiff     = NewNode("audio/aiff", "aiff", match_Aiff)
	Au       = NewNode("audio/basic", "au", match_Au)
	Amr      = NewNode("audio/amr", "amr", match_Amr)

	Mp4       = NewNode("video/mp4", "mp4", match_Mp4)
	WebM      = NewNode("video/webm", "webm", match_WebM)
	Mpeg      = NewNode("video/mpeg", "mpeg", match_Mpeg)
	QuickTime = NewNode("video/quicktime", "mov", match_QuickTime)
	ThreeGP   = NewNode("video/3gp", "3gp", match_ThreeGP)
	Avi       = NewNode("video/x-msvideo", "avi", match_Avi)
	Flv       = NewNode("video/x-flv", "flv", match_Flv)
	Mkv       = NewNode("video/x-matroska", "mkv", match_Mkv)
)

var fileicons = map[string]string{
	".7z":       "",
	".ai":       "",
	".apk":      "",
	".avi":      "",
	".bat":      "",
	".bmp":      "",
	".bz2":      "",
	".c":        "",
	".c++":      "",
	".cab":      "",
	".cc":       "",
	".clj":      "",
	".cljc":     "",
	".cljs":     "",
	".coffee":   "",
	".conf":     "",
	".cp":       "",
	".cpio":     "",
	".cpp":      "",
	".css":      "",
	".cxx":      "",
	".d":        "",
	".dart":     "",
	".db":       "",
	".deb":      "",
	".diff":     "",
	".dump":     "",
	".edn":      "",
	".ejs":      "",
	".epub":     "",
	".erl":      "",
	".f#":       "",
	".fish":     "",
	".flac":     "",
	".flv":      "",
	".fs":       "",
	".fsi":      "",
	".fsscript": "",
	".fsx":      "",
	".gem":      "",
	".gif":      "",
	".go":       "",
	".gz":       "",
	".gzip":     "",
	".hbs":      "",
	".hrl":      "",
	".hs":       "",
	".htm":      "",
	".html":     "",
	".ico":      "",
	".ini":      "",
	".java":     "",
	".jl":       "",
	".jpeg":     "",
	".jpg":      "",
	".js":       "",
	".json":     "",
	".jsx":      "",
	".less":     "",
	".lha":      "",
	".lhs":      "",
	".log":      "",
	".lua":      "",
	".lzh":      "",
	".lzma":     "",
	".markdown": "",
	".md":       "",
	".mkv":      "",
	".ml":       "λ",
	".mli":      "λ",
	".mov":      "",
	".mp3":      "",
	".mp4":      "",
	".mpeg":     "",
	".mpg":      "",
	".mustache": "",
	".ogg":      "",
	".pdf":      "",
	".php":      "",
	".pl":       "",
	".pm":       "",
	".png":      "",
	".psb":      "",
	".psd":      "",
	".py":       "",
	".pyc":      "",
	".pyd":      "",
	".pyo":      "",
	".rar":      "",
	".rb":       "",
	".rc":       "",
	".rlib":     "",
	".rpm":      "",
	".rs":       "",
	".rss":      "",
	".scala":    "",
	".scss":     "",
	".sh":       "",
	".slim":     "",
	".sln":      "",
	".sql":      "",
	".styl":     "",
	".suo":      "",
	".t":        "",
	".tar":      "",
	".tgz":      "",
	".ts":       "",
	".twig":     "",
	".vim":      "",
	".vimrc":    "",
	".wav":      "",
	".xml":      "",
	".xul":      "",
	".xz":       "",
	".yml":      "",
	".zip":      "",
}

var categoryicons = map[string]string{
	"folder/folder": "",
	"file/default":  "",
}

// Zip matches a zip archive.
func match_Zip(in []byte) bool {
	return len(in) > 3 &&
		in[0] == 0x50 && in[1] == 0x4B &&
		(in[2] == 0x3 || in[2] == 0x5 || in[2] == 0x7) &&
		(in[3] == 0x4 || in[3] == 0x6 || in[3] == 0x8)
}

// SevenZ matches a 7z archive.
func match_SevenZ(in []byte) bool {
	return bytes.Equal(in[:6], []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C})
}

// Epub matches an EPUB file.
func match_Epub(in []byte) bool {
	if len(in) < 58 {
		return false
	}
	in = in[30:58]

	return bytes.Equal(in, []byte("mimetypeapplication/epub+zip"))
}

// Jar matches a Java archive file.
func match_Jar(in []byte) bool {
	return bytes.Contains(in, []byte("META-INF/MANIFEST.MF"))
}

// Gzip matched gzip files based on http://www.zlib.org/rfc-gzip.html#header-trailer
func match_Gzip(in []byte) bool {
	return bytes.Equal(in[:2], []byte{0x1f, 0x8b})
}

func match_Mp3(in []byte) bool {
	return bytes.HasPrefix(in, []byte("\x49\x44\x33"))
}

// Flac matches a Free Lossless Audio Codec file.
func match_Flac(in []byte) bool {
	return bytes.HasPrefix(in, []byte("\x66\x4C\x61\x43\x00\x00\x00\x22"))
}

// Midi matches a Musical Instrument Digital Interface file.
func match_Midi(in []byte) bool {
	return bytes.HasPrefix(in, []byte("\x4D\x54\x68\x64"))
}

// Ape matches a Monkey's Audio file.
func match_Ape(in []byte) bool {
	return bytes.HasPrefix(in, []byte("\x4D\x41\x43\x20\x96\x0F\x00\x00\x34\x00\x00\x00\x18\x00\x00\x00\x90\xE3"))
}

// Musepack matches a Musepack file.
func match_MusePack(in []byte) bool {
	return bytes.Equal(in[:4], []byte("MPCK"))
}

// Wav matches a Waveform Audio File Format file.
func match_Wav(in []byte) bool {
	return bytes.Equal(in[:4], []byte("\x52\x49\x46\x46")) &&
		bytes.Equal(in[8:12], []byte("\x57\x41\x56\x45"))
}

// Aiff matches Audio Interchange File Format file.
func match_Aiff(in []byte) bool {
	return bytes.Equal(in[:4], []byte("\x46\x4F\x52\x4D")) &&
		bytes.Equal(in[8:12], []byte("\x41\x49\x46\x46"))
}

// Ogg matches an Ogg file.
func match_Ogg(in []byte) bool {
	return bytes.Equal(in[:5], []byte("\x4F\x67\x67\x53\x00"))
}

// Au matches a Sun Microsystems au file.
func match_Au(in []byte) bool {
	return bytes.Equal(in[:4], []byte("\x2E\x73\x6E\x64"))
}

// Amr matches an Adaptive Multi-Rate file.
func match_Amr(in []byte) bool {
	return bytes.Equal(in[:5], []byte("\x23\x21\x41\x4D\x52"))
}

func match_Pdf(in []byte) bool {
	return bytes.Equal(in[:4], []byte{0x25, 0x50, 0x44, 0x46})
}

// Png matches a Portable Network Graphics file.
func match_Png(in []byte) bool {
	return bytes.Equal(in[:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
}

// Jpg matches a Joint Photographic Experts Group file.
func match_Jpg(in []byte) bool {
	return bytes.Equal(in[:3], []byte{0xFF, 0xD8, 0xFF})
}

// Gif matches a Graphics Interchange Format file.
func match_Gif(in []byte) bool {
	return bytes.HasPrefix(in, []byte("GIF87a")) ||
		bytes.HasPrefix(in, []byte("GIF89a"))
}

// Webp matches a WebP file.
func match_Webp(in []byte) bool {
	return len(in) > 11 &&
		bytes.Equal(in[0:4], []byte{0x52, 0x49, 0x46, 0x46}) &&
		bytes.Equal(in[8:12], []byte{0x57, 0x45, 0x42, 0x50})
}

// Bmp matches a bitmap image file.
func match_Bmp(in []byte) bool {
	return len(in) > 1 &&
		in[0] == 0x42 &&
		in[1] == 0x4D
}

// Ps matches a PostScript file.
func match_Ps(in []byte) bool {
	return bytes.HasPrefix(in, []byte("%!PS-Adobe-"))
}

// Psd matches a Photoshop Document file.
func match_Psd(in []byte) bool {
	return bytes.HasPrefix(in, []byte("8BPS"))
}

// Ico matches an ICO file.
func match_Ico(in []byte) bool {
	return len(in) > 3 &&
		in[0] == 0x00 && in[1] == 0x00 &&
		in[2] == 0x01 && in[3] == 0x00
}

// Tiff matches a Tagged Image File Format file.
func match_Tiff(in []byte) bool {
	return bytes.Equal(in[:4], []byte{0x49, 0x49, 0x2A, 0x00}) ||
		bytes.Equal(in[:4], []byte{0x4D, 0x4D, 0x00, 0x2A})
}

// Xlsx matches a Microsoft Excel 2007 file.
func match_Xlsx(in []byte) bool {
	return bytes.Contains(in, []byte("xl/"))
}

// Docx matches a Microsoft Office 2007 file.
func match_Docx(in []byte) bool {
	return bytes.Contains(in, []byte("word/"))
}

// Pptx matches a Microsoft PowerPoint 2007 file.
func match_Pptx(in []byte) bool {
	return bytes.Contains(in, []byte("ppt/"))
}

// Mp4 matches an MP4 file.
func match_Mp4(in []byte) bool {
	if len(in) < 12 {
		return false
	}

	mp4ftype := []byte("ftyp")
	mp4 := []byte("mp4")
	boxSize := int(binary.BigEndian.Uint32(in[:4]))
	if boxSize%4 != 0 || len(in) < boxSize {
		return false
	}
	if !bytes.Equal(in[4:8], mp4ftype) {
		return false
	}
	for st := 8; st < boxSize; st += 4 {
		if st == 12 {
			// minor version number
			continue
		}
		if bytes.Equal(in[st:st+3], mp4) {
			return true
		}
	}

	return false
}

// WebM matches a WebM file.
func match_WebM(in []byte) bool {
	return isMatroskaFileTypeMatched(in, "webm")
}

// Mkv matches a mkv file
func match_Mkv(in []byte) bool {
	return isMatroskaFileTypeMatched(in, "matroska")
}

// isMatroskaFileTypeMatched is used for webm and mkv file matching.
// It checks for .Eß£ sequence. If the sequence is found,
// then it means it is Matroska media container, including WebM.
// Then it verifies which of the file type it is representing by matching the
// file specific string.
func isMatroskaFileTypeMatched(in []byte, flType string) bool {
	if bytes.HasPrefix(in, []byte("\x1A\x45\xDF\xA3")) {
		return isFileTypeNamePresent(in, flType)
	}
	return false
}

// isFileTypeNamePresent accepts the matroska input data stream and searches
// for the given file type in the stream. Return whether a match is found.
// The logic of search is: find first instance of \x42\x82 and then
// search for given string after one byte of above instance.
func isFileTypeNamePresent(in []byte, flType string) bool {
	var ind int
	if len(in) >= 4096 { // restricting length to 4096
		ind = bytes.Index(in[0:4096], []byte("\x42\x82"))
	} else {
		ind = bytes.Index(in, []byte("\x42\x82"))
	}
	if ind > 0 {
		// filetype name will be present exactly
		// one byte after the match of the two bytes "\x42\x82"
		return bytes.HasPrefix(in[ind+3:], []byte(flType))
	}
	return false
}

// ThreeGP matches a Third Generation Partnership Project file.
func match_ThreeGP(in []byte) bool {
	return len(in) > 11 &&
		bytes.HasPrefix(in[4:], []byte("\x66\x74\x79\x70\x33\x67\x70"))
}

// Flv matches a Flash video file.
func match_Flv(in []byte) bool {
	return bytes.HasPrefix(in, []byte("\x46\x4C\x56\x01"))
}

// Mpeg matches a Moving Picture Experts Group file.
func match_Mpeg(in []byte) bool {
	return len(in) > 3 && bytes.Equal(in[:3], []byte("\x00\x00\x01")) &&
		in[3] >= 0xB0 && in[3] <= 0xBF
}

// QuickTime matches a QuickTime File Format file.
func match_QuickTime(in []byte) bool {
	return len(in) > 12 &&
		(bytes.Equal(in[4:12], []byte("ftypqt  ")) ||
			bytes.Equal(in[4:8], []byte("moov")))
}

// Avi matches an Audio Video Interleaved file.
func match_Avi(in []byte) bool {
	return len(in) > 16 &&
		bytes.Equal(in[:4], []byte("RIFF")) &&
		bytes.Equal(in[8:16], []byte("AVI LIST"))
}

type (
	markupSig  []byte
	ciSig      []byte // case insensitive signature
	shebangSig []byte // matches !# followed by the signature
	sig        interface {
		detect([]byte) bool
	}
)

var (
	htmlSigs = []sig{
		markupSig("<!DOCTYPE HTML"),
		markupSig("<HTML"),
		markupSig("<HEAD"),
		markupSig("<SCRIPT"),
		markupSig("<IFRAME"),
		markupSig("<H1"),
		markupSig("<DIV"),
		markupSig("<FONT"),
		markupSig("<TABLE"),
		markupSig("<A"),
		markupSig("<STYLE"),
		markupSig("<TITLE"),
		markupSig("<B"),
		markupSig("<BODY"),
		markupSig("<BR"),
		markupSig("<P"),
		markupSig("<!--"),
	}
	xmlSigs = []sig{
		markupSig("<?XML"),
	}
	phpSigs = []sig{
		ciSig("<?PHP"),
		ciSig("<?\n"),
		ciSig("<?\r"),
		ciSig("<? "),
		shebangSig("/usr/local/bin/php"),
		shebangSig("/usr/bin/php"),
		shebangSig("/usr/bin/env php"),
	}
	jsSigs = []sig{
		shebangSig("/bin/node"),
		shebangSig("/usr/bin/node"),
		shebangSig("/bin/nodejs"),
		shebangSig("/usr/bin/nodejs"),
		shebangSig("/usr/bin/env node"),
		shebangSig("/usr/bin/env nodejs"),
	}
	luaSigs = []sig{
		shebangSig("/usr/bin/lua"),
		shebangSig("/usr/local/bin/lua"),
		shebangSig("/usr/bin/env lua"),
	}
	perlSigs = []sig{
		shebangSig("/usr/bin/perl"),
		shebangSig("/usr/bin/env perl"),
		shebangSig("/usr/bin/env perl"),
	}
	pythonSigs = []sig{
		shebangSig("/usr/bin/python"),
		shebangSig("/usr/local/bin/python"),
		shebangSig("/usr/bin/env python"),
		shebangSig("/usr/bin/env python"),
	}
)

// Txt matches a text file.
func match_Txt(in []byte) bool {
	in = trimLWS(in)
	for _, b := range in {
		if b <= 0x08 ||
			b == 0x0B ||
			0x0E <= b && b <= 0x1A ||
			0x1C <= b && b <= 0x1F {
			return false
		}
	}

	return true
}

func detect(in []byte, sigs []sig) bool {
	for _, sig := range sigs {
		if sig.detect(in) {
			return true
		}
	}

	return false
}

// Html matches a Hypertext Markup Language file.
func match_Html(in []byte) bool {
	return detect(in, htmlSigs)
}

// Xml matches an Extensible Markup Language file.
func match_Xml(in []byte) bool {
	return detect(in, xmlSigs)
}

// Php matches a PHP: Hypertext Preprocessor file.
func match_Php(in []byte) bool {
	return detect(in, phpSigs)
}

// Json matches a JavaScript Object Notation file.
func match_Json(in []byte) bool {
	parsed, err := json.Scan(in)
	if len(in) < ReadLimit {
		return err == nil
	}

	return parsed == len(in)
}

// Js matches a Javascript file.
func match_Js(in []byte) bool {
	return detect(in, jsSigs)
}

// Lua matches a Lua programming language file.
func match_Lua(in []byte) bool {
	return detect(in, luaSigs)
}

// Perl matches a Perl programming language file.
func match_Perl(in []byte) bool {
	return detect(in, perlSigs)
}

// Python matches a Python programming language file.
func match_Python(in []byte) bool {
	return detect(in, pythonSigs)
}

// Implement sig interface.
func (hSig markupSig) detect(in []byte) bool {
	if len(in) < len(hSig)+1 {
		return false
	}

	// perform case insensitive check
	for i, b := range hSig {
		db := in[i]
		if 'A' <= b && b <= 'Z' {
			db &= 0xDF
		}
		if b != db {
			return false
		}
	}
	// Next byte must be space or right angle bracket.
	if db := in[len(hSig)]; db != ' ' && db != '>' {
		return false
	}

	return true
}

// Implement sig interface.
func (tSig ciSig) detect(in []byte) bool {
	if len(in) < len(tSig)+1 {
		return false
	}

	// perform case insensitive check
	for i, b := range tSig {
		db := in[i]
		if 'A' <= b && b <= 'Z' {
			db &= 0xDF
		}
		if b != db {
			return false
		}
	}

	return true
}

// a valid shebang starts with the "#!" characters
// followed by any number of spaces
// followed by the path to the interpreter and optionally, the args for the interpreter
func (sSig shebangSig) detect(in []byte) bool {
	in = firstLine(in)

	if len(in) < len(sSig)+2 {
		return false
	}
	if in[0] != '#' || in[1] != '!' {
		return false
	}

	in = trimLWS(trimRWS(in[2:]))

	return bytes.Equal(in, sSig)
}

// Rtf matches a Rich Text Format file.
func match_Rtf(in []byte) bool {
	return bytes.Equal(in[:6], []byte("\x7b\x5c\x72\x74\x66\x31"))
}

const ReadLimit = 520

// True is a dummy matching function used to match any input.
func match_True(_ []byte) bool {
	return true
}

// False is a dummy matching function used to never match input.
func match_False(_ []byte) bool {
	return false
}

// trimLWS trims whitespace from beginning of the input.
func trimLWS(in []byte) []byte {
	firstNonWS := 0
	for ; firstNonWS < len(in) && isWS(in[firstNonWS]); firstNonWS++ {
	}

	return in[firstNonWS:]
}

// trimRWS trims whitespace from the end of the input.
func trimRWS(in []byte) []byte {
	lastNonWS := len(in) - 1
	for ; lastNonWS > 0 && isWS(in[lastNonWS]); lastNonWS-- {
	}

	return in[:lastNonWS+1]
}

// firstLine returns the
func firstLine(in []byte) []byte {
	lineEnd := 0
	for ; lineEnd < len(in) && in[lineEnd] != '\n'; lineEnd++ {
	}

	return in[:lineEnd]
}

func isWS(b byte) bool {
	return b == '\t' || b == '\n' || b == '\x0c' || b == '\r' || b == ' '
}
