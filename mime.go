//based on https://github.com/gabriel-vasile/mimetype

package dirk

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/gabriel-vasile/mimetype/matchers"
)

// Detect returns the mime type and extension of the provided byte slice.
func Detect(in []byte) (mime, extension string) {
	n := Root.match(in, Root)
	return n.Mime(), n.Extension()
}

// DetectReader returns the mime type and extension of the byte slice read
// from the provided reader.
func DetectReader(r io.Reader) (mime, extension string, err error) {
	in := make([]byte, matchers.ReadLimit)
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

var Root = NewNode("application/octet-stream", "", matchers.True,
	SevenZ, Zip, Pdf, Doc, Xls, Ppt, Ps, Psd, Ogg,
	Png, Jpg, Gif, Webp, Tiff, Bmp, Ico,
	Mp3, Flac, Midi, Ape, MusePack, Amr, Wav, Aiff, Au,
	Mpeg, QuickTime, Mp4, WebM, ThreeGP, Avi, Flv, Mkv,
	Txt, Gzip,
)

// The list of nodes appended to the Root node
var (
	Gzip   = NewNode("application/gzip", "gz", matchers.Gzip)
	SevenZ = NewNode("application/x-7z-compressed", "7z", match_SevenZ)
	Zip    = NewNode("application/zip", "zip", match_Zip, Xlsx, Docx, Pptx, Epub, Jar)
	Pdf    = NewNode("application/pdf", "pdf", matchers.Pdf)
	Xlsx   = NewNode("application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx", matchers.Xlsx)
	Docx   = NewNode("application/vnd.openxmlformats-officedocument.wordprocessingml.document", "docx", matchers.Docx)
	Pptx   = NewNode("application/vnd.openxmlformats-officedocument.presentationml.presentation", "pptx", matchers.Pptx)
	Epub   = NewNode("application/epub+zip", "epub", matchers.Epub)
	Jar    = NewNode("application/jar", "jar", matchers.Jar, Apk)
	Apk    = NewNode("application/vnd.android.package-archive", "apk", matchers.False)
	Doc    = NewNode("application/msword", "doc", matchers.Doc)
	Ppt    = NewNode("application/vnd.ms-powerpoint", "ppt", matchers.Ppt)
	Xls    = NewNode("application/vnd.ms-excel", "xls", matchers.Xls)
	Ps     = NewNode("application/postscript", "ps", matchers.Ps)
	Psd    = NewNode("application/x-photoshop", "psd", matchers.Psd)
	Ogg    = NewNode("application/ogg", "ogg", matchers.Ogg)

	Txt  = NewNode("text/plain", "txt", matchers.Txt, Html, Xml, Php, Js, Lua, Perl, Python, Json, Rtf)
	Xml  = NewNode("text/xml; charset=utf-8", "xml", matchers.Xml, Svg, X3d, Kml, Collada, Gml, Gpx)
	Json = NewNode("application/json", "json", matchers.Json)
	Html = NewNode("text/html; charset=utf-8", "html", matchers.Html)
	Php  = NewNode("text/x-php; charset=utf-8", "php", matchers.Php)
	Rtf  = NewNode("text/rtf", "rtf", matchers.Rtf)

	Js     = NewNode("application/javascript", "js", matchers.Js)
	Lua    = NewNode("text/x-lua", "lua", matchers.Lua)
	Perl   = NewNode("text/x-perl", "pl", matchers.Perl)
	Python = NewNode("application/x-python", "py", matchers.Python)

	Svg     = NewNode("image/svg+xml", "svg", matchers.False)
	X3d     = NewNode("model/x3d+xml", "x3d", matchers.False)
	Kml     = NewNode("application/vnd.google-earth.kml+xml", "kml", matchers.False)
	Collada = NewNode("model/vnd.collada+xml", "dae", matchers.False)
	Gml     = NewNode("application/gml+xml", "gml", matchers.False)
	Gpx     = NewNode("application/gpx+xml", "gpx", matchers.False)

	Png  = NewNode("image/png", "png", matchers.Png)
	Jpg  = NewNode("image/jpeg", "jpg", matchers.Jpg)
	Gif  = NewNode("image/gif", "gif", matchers.Gif)
	Webp = NewNode("image/webp", "webp", matchers.Webp)
	Tiff = NewNode("image/tiff", "tiff", matchers.Tiff)
	Bmp  = NewNode("image/bmp", "bmp", matchers.Bmp)
	Ico  = NewNode("image/x-icon", "ico", matchers.Ico)

	Mp3      = NewNode("audio/mpeg", "mp3", matchers.Mp3)
	Flac     = NewNode("audio/flac", "flac", matchers.Flac)
	Midi     = NewNode("audio/midi", "midi", matchers.Midi)
	Ape      = NewNode("audio/ape", "ape", matchers.Ape)
	MusePack = NewNode("audio/musepack", "mpc", matchers.MusePack)
	Wav      = NewNode("audio/wav", "wav", matchers.Wav)
	Aiff     = NewNode("audio/aiff", "aiff", matchers.Aiff)
	Au       = NewNode("audio/basic", "au", matchers.Au)
	Amr      = NewNode("audio/amr", "amr", matchers.Amr)

	Mp4       = NewNode("video/mp4", "mp4", matchers.Mp4)
	WebM      = NewNode("video/webm", "webm", matchers.WebM)
	Mpeg      = NewNode("video/mpeg", "mpeg", matchers.Mpeg)
	QuickTime = NewNode("video/quicktime", "mov", matchers.QuickTime)
	ThreeGP   = NewNode("video/3gp", "3gp", matchers.ThreeGP)
	Avi       = NewNode("video/x-msvideo", "avi", matchers.Avi)
	Flv       = NewNode("video/x-flv", "flv", matchers.Flv)
	Mkv       = NewNode("video/x-matroska", "mkv", matchers.Mkv)
)

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
