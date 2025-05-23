package ui

import (
	"strings"
)

// GetIconForFile returns an appropriate icon based on file name and extension
func GetIconForFile(name, ext string) string {
	// default icon for all files. try to find a better one though...
	icon := icons["file"]

	// resolve aliased extensions
	extKey := strings.ToLower(ext)
	alias, hasAlias := aliases[extKey]
	if hasAlias {
		extKey = alias
	}

	// see if we can find a better icon based on extension alone
	betterIcon, hasBetterIcon := icons[extKey]
	if hasBetterIcon {
		icon = betterIcon
	}

	// now look for icons based on full names
	fullName := name
	if ext != "" {
		fullName += "." + ext
	}

	fullName = strings.ToLower(fullName)
	fullAlias, hasFullAlias := aliases[fullName]
	if hasFullAlias {
		fullName = fullAlias
	}
	bestIcon, hasBestIcon := icons[fullName]
	if hasBestIcon {
		icon = bestIcon
	}
	return icon
}

// GetIconForFolder returns an appropriate icon for a folder
func GetIconForFolder(name string) string {
	icon := folders["folder"]
	betterIcon, hasBetterIcon := folders[name]
	if hasBetterIcon {
		icon = betterIcon
	}
	return icon
}

// icons maps file types to their corresponding icons
var icons = map[string]string{
	"ai":           "\ue7b4",
	"android":      "\ue70e",
	"apple":        "\uf179",
	"as":           "\ue60b",
	"asm":          "󰘚",
	"audio":        "\uf1c7",
	"avro":         "\ue60b",
	"bf":           "\uf067",
	"binary":       "\uf471",
	"bzl":          "\ue63a",
	"c":            "\ue61e",
	"cfg":          "\uf423",
	"clj":          "\ue768",
	"coffee":       "\ue751",
	"conf":         "\ue615",
	"cpp":          "\ue61d",
	"cfm":          "\ue645",
	"cr":           "\ue62f",
	"cs":           "\ue648",
	"cson":         "\ue601",
	"css":          "\ue749",
	"cu":           "\ue64b",
	"d":            "\ue7af",
	"dart":         "\ue64c",
	"db":           "\uf1c0",
	"deb":          "\uf306",
	"diff":         "\uf440",
	"doc":          "\uf1c2",
	"dockerfile":   "\ue650",
	"dpkg":         "\uf17c",
	"ebook":        "\uf02d",
	"elm":          "\ue62c",
	"env":          "\uf462",
	"erl":          "\ue7b1",
	"ex":           "\ue62d",
	"f":            "󱈚",
	"file":         "\uf15b",
	"font":         "\uf031",
	"fs":           "\ue7a7",
	"gb":           "\ue272",
	"gform":        "\uf298",
	"git":          "\ue702",
	"go":           "\ue724",
	"graphql":      "\ue662",
	"gleam":        "★",
	"glp":          "󰆧",
	"groovy":       "\ue775",
	"gruntfile.js": "\ue74c",
	"gulpfile.js":  "\ue610",
	"gv":           "\ue225",
	"h":            "\uf0fd",
	"haml":         "\ue664",
	"hs":           "\ue777",
	"html":         "\uf13b",
	"hx":           "\ue666",
	"ics":          "\uf073",
	"image":        "\uf1c5",
	"iml":          "\ue7b5",
	"ini":          "󰅪",
	"ino":          "\ue255",
	"iso":          "󰋊",
	"jade":         "\ue66c",
	"java":         "\ue738",
	"jenkinsfile":  "\ue767",
	"jl":           "\ue624",
	"js":           "\ue781",
	"json":         "\ue60b",
	"jsx":          "\ue7ba",
	"key":          "\uf43d",
	"ko":           "\uebc6",
	"kt":           "\ue634",
	"less":         "\ue758",
	"lock":         "\uf023",
	"log":          "\uf18d",
	"lua":          "\ue620",
	"maintainers":  "\uf0c0",
	"makefile":     "\ue20f",
	"md":           "\uf48a",
	"mjs":          "\ue718",
	"ml":           "󰘧",
	"mustache":     "\ue60f",
	"nc":           "󰋁",
	"nim":          "\ue677",
	"nix":          "\uf313",
	"npmignore":    "\ue71e",
	"package":      "󰏗",
	"passwd":       "\uf023",
	"patch":        "\uf440",
	"pdf":          "\uf1c1",
	"php":          "\ue608",
	"pl":           "\ue7a1",
	"prisma":       "\ue684",
	"ppt":          "\uf1c4",
	"psd":          "\ue7b8",
	"pwsh":         "󰨊",
	"py":           "\ue606",
	"r":            "\ue68a",
	"rb":           "\ue21e",
	"rdb":          "\ue76d",
	"rpm":          "\uf17c",
	"rs":           "\ue7a8",
	"rss":          "\uf09e",
	"rst":          "󰅫",
	"rubydoc":      "\ue73b",
	"sass":         "\ue603",
	"scala":        "\ue737",
	"shell":        "\uf489",
	"shp":          "󰙞",
	"sol":          "󰡪",
	"sqlite":       "\ue7c4",
	"styl":         "\ue600",
	"svelte":       "\ue697",
	"swift":        "\ue755",
	"tex":          "\u222b",
	"tf":           "\ue69a",
	"toml":         "󰅪",
	"ts":           "󰛦",
	"twig":         "\ue61c",
	"txt":          "\uf15c",
	"vagrantfile":  "\ue21e",
	"video":        "\uf03d",
	"vim":          "\ue62b",
	"vue":          "\ue6a0",
	"windows":      "\uf17a",
	"xls":          "\uf1c3",
	"xml":          "\ue796",
	"yml":          "\ue601",
	"zig":          "\ue6a9",
	"zip":          "\uf410",
}

// aliases maps file extensions and names to their canonical types
var aliases = map[string]string{
	"apk":              "android",
	"gradle":           "android",
	"ds_store":         "apple",
	"localized":        "apple",
	"m":                "apple",
	"mm":               "apple",
	"s":                "asm",
	"aac":              "audio",
	"alac":             "audio",
	"flac":             "audio",
	"m4a":              "audio",
	"mka":              "audio",
	"mp3":              "audio",
	"ogg":              "audio",
	"opus":             "audio",
	"wav":              "audio",
	"wma":              "audio",
	"b":                "bf",
	"bson":             "binary",
	"feather":          "binary",
	"mat":              "binary",
	"o":                "binary",
	"pb":               "binary",
	"pickle":           "binary",
	"pkl":              "binary",
	"tfrecord":         "binary",
	"conf":             "cfg",
	"config":           "cfg",
	"cljc":             "clj",
	"cljs":             "clj",
	"editorconfig":     "conf",
	"rc":               "conf",
	"c++":              "cpp",
	"cc":               "cpp",
	"cxx":              "cpp",
	"scss":             "css",
	"sql":              "db",
	"docx":             "doc",
	"gdoc":             "doc",
	"dockerignore":     "dockerfile",
	"epub":             "ebook",
	"ipynb":            "ebook",
	"mobi":             "ebook",
	"f03":              "f",
	"f77":              "f",
	"f90":              "f",
	"f95":              "f",
	"for":              "f",
	"fpp":              "f",
	"ftn":              "f",
	"eot":              "font",
	"otf":              "font",
	"ttf":              "font",
	"woff":             "font",
	"woff2":            "font",
	"fsi":              "fs",
	"fsscript":         "fs",
	"fsx":              "fs",
	"dna":              "gb",
	"gitattributes":    "git",
	"gitconfig":        "git",
	"gitignore":        "git",
	"gitignore_global": "git",
	"gitmirrorall":     "git",
	"gitmodules":       "git",
	"gltf":             "glp",
	"gsh":              "groovy",
	"gvy":              "groovy",
	"gy":               "groovy",
	"h++":              "h",
	"hh":               "h",
	"hpp":              "h",
	"hxx":              "h",
	"lhs":              "hs",
	"htm":              "html",
	"xhtml":            "html",
	"bmp":              "image",
	"cbr":              "image",
	"cbz":              "image",
	"dvi":              "image",
	"eps":              "image",
	"gif":              "image",
	"ico":              "image",
	"jpeg":             "image",
	"jpg":              "image",
	"nef":              "image",
	"orf":              "image",
	"pbm":              "image",
	"pgm":              "image",
	"png":              "image",
	"pnm":              "image",
	"ppm":              "image",
	"pxm":              "image",
	"sixel":            "image",
	"stl":              "image",
	"svg":              "image",
	"tif":              "image",
	"tiff":             "image",
	"webp":             "image",
	"xpm":              "image",
	"disk":             "iso",
	"dmg":              "iso",
	"img":              "iso",
	"ipsw":             "iso",
	"smi":              "iso",
	"vhd":              "iso",
	"vhdx":             "iso",
	"vmdk":             "iso",
	"jar":              "java",
	"cjs":              "js",
	"properties":       "json",
	"webmanifest":      "json",
	"tsx":              "jsx",
	"cjsx":             "jsx",
	"cer":              "key",
	"crt":              "key",
	"der":              "key",
	"gpg":              "key",
	"p7b":              "key",
	"pem":              "key",
	"pfx":              "key",
	"pgp":              "key",
	"license":          "key",
	"codeowners":       "maintainers",
	"credits":          "maintainers",
	"cmake":            "makefile",
	"justfile":         "makefile",
	"markdown":         "md",
	"mkd":              "md",
	"rdoc":             "md",
	"readme":           "md",
	"mli":              "ml",
	"sml":              "ml",
	"netcdf":           "nc",
	"brewfile":         "package",
	"cargo.toml":       "package",
	"cargo.lock":       "package",
	"go.mod":           "package",
	"go.sum":           "package",
	"pyproject.toml":   "package",
	"poetry.lock":      "package",
	"package.json":     "package",
	"pipfile":          "package",
	"pipfile.lock":     "package",
	"php3":             "php",
	"php4":             "php",
	"php5":             "php",
	"phpt":             "php",
	"phtml":            "php",
	"gslides":          "ppt",
	"pptx":             "ppt",
	"ps1":              "pwsh",
	"psd1":             "pwsh",
	"psm1":             "pwsh",
	"pxd":              "py",
	"pyc":              "py",
	"pyx":              "py",
	"whl":              "py",
	"rdata":            "r",
	"rds":              "r",
	"rmd":              "r",
	"gemfile":          "rb",
	"gemspec":          "rb",
	"guardfile":        "rb",
	"procfile":         "rb",
	"rakefile":         "rb",
	"rspec":            "rb",
	"rspec_parallel":   "rb",
	"rspec_status":     "rb",
	"ru":               "rb",
	"erb":              "rubydoc",
	"slim":             "rubydoc",
	"awk":              "shell",
	"bash":             "shell",
	"bash_history":     "shell",
	"bash_profile":     "shell",
	"bashrc":           "shell",
	"csh":              "shell",
	"fish":             "shell",
	"ksh":              "shell",
	"sh":               "shell",
	"zsh":              "shell",
	"zsh-theme":        "shell",
	"zshrc":            "shell",
	"plpgsql":          "sql",
	"plsql":            "sql",
	"psql":             "sql",
	"tsql":             "sql",
	"sl3":              "sqlite",
	"sqlite3":          "sqlite",
	"stylus":           "styl",
	"cls":              "tex",
	"avi":              "video",
	"flv":              "video",
	"m2v":              "video",
	"mkv":              "video",
	"mov":              "video",
	"mp4":              "video",
	"mpeg":             "video",
	"mpg":              "video",
	"ogm":              "video",
	"ogv":              "video",
	"vob":              "video",
	"webm":             "video",
	"vimrc":            "vim",
	"bat":              "windows",
	"cmd":              "windows",
	"exe":              "windows",
	"csv":              "xls",
	"gsheet":           "xls",
	"xlsx":             "xls",
	"plist":            "xml",
	"xul":              "xml",
	"yaml":             "yml",
	"7z":               "zip",
	"Z":                "zip",
	"bz2":              "zip",
	"gz":               "zip",
	"lzma":             "zip",
	"par":              "zip",
	"rar":              "zip",
	"tar":              "zip",
	"tc":               "zip",
	"tgz":              "zip",
	"txz":              "zip",
	"xz":               "zip",
	"z":                "zip",
}

// folders maps folder names to their corresponding icons
var folders = map[string]string{
	".atom":                 "\ue764",
	".aws":                  "\ue7ad",
	".docker":               "\ue7b0",
	".gem":                  "\ue21e",
	".git":                  "\ue5fb",
	".git-credential-cache": "\ue5fb",
	".github":               "\ue5fd",
	".npm":                  "\ue5fa",
	".nvm":                  "\ue718",
	".rvm":                  "\ue21e",
	".Trash":                "\uf1f8",
	".vscode":               "\ue70c",
	".vim":                  "\ue62b",
	"config":                "\ue5fc",
	"folder":                "\uf07c",
	"hidden":                "\uf023",
	"node_modules":          "\ue5fa",
}

// otherIcons maps special file types to their corresponding icons
var otherIcons = map[string]string{
	"link":       "\uf0c1",
	"linkDir":    "\uf0c1",
	"brokenLink": "\uf127",
	"device":     "\uf0a0",
	"socket":     "\uf1e6",
	"pipe":       "\ufce3",
}
