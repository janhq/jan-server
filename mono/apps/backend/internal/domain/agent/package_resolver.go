package agent

import (
	"strings"
)

// moduleToPackage maps Python module names to their pip package names.
// This handles cases where the import name differs from the package name.
var moduleToPackage = map[string]string{
	// Data Science & ML
	"sklearn":  "scikit-learn",
	"skimage":  "scikit-image",
	"cv2":      "opencv-python",
	"PIL":      "Pillow",
	"Image":    "Pillow", // from PIL import Image
	"scipy":    "scipy",
	"numpy":    "numpy",
	"pandas":   "pandas",
	"torch":    "torch",
	"tf":       "tensorflow",
	"keras":    "keras",
	"xgboost":  "xgboost",
	"lightgbm": "lightgbm",
	"catboost": "catboost",

	// Web & Parsing
	"bs4":        "beautifulsoup4",
	"lxml":       "lxml",
	"html5lib":   "html5lib",
	"selenium":   "selenium",
	"playwright": "playwright",

	// Config & Utils
	"yaml":     "PyYAML",
	"dotenv":   "python-dotenv",
	"dateutil": "python-dateutil",
	"tz":       "pytz",
	"pytz":     "pytz",

	// Auth & Security
	"jwt":          "PyJWT",
	"cryptography": "cryptography",
	"nacl":         "pynacl",
	"bcrypt":       "bcrypt",

	// Database
	"psycopg2":   "psycopg2-binary",
	"MySQLdb":    "mysqlclient",
	"pymysql":    "PyMySQL",
	"pymongo":    "pymongo",
	"redis":      "redis",
	"sqlalchemy": "SQLAlchemy",

	// Cloud & APIs
	"boto3":           "boto3",
	"botocore":        "botocore",
	"google":          "google-api-python-client",
	"googleapiclient": "google-api-python-client",
	"azure":           "azure-storage-blob",

	// HTTP & Async
	"requests":   "requests",
	"httpx":      "httpx",
	"aiohttp":    "aiohttp",
	"urllib3":    "urllib3",
	"websockets": "websockets",
	"httplib2":   "httplib2",

	// Data Formats
	"msgpack":    "msgpack",
	"orjson":     "orjson",
	"ujson":      "ujson",
	"simplejson": "simplejson",
	"toml":       "toml",
	"tomllib":    "tomli", // Python < 3.11

	// Visualization
	"matplotlib": "matplotlib",
	"seaborn":    "seaborn",
	"plotly":     "plotly",
	"bokeh":      "bokeh",
	"altair":     "altair",

	// NLP
	"nltk":         "nltk",
	"spacy":        "spacy",
	"transformers": "transformers",
	"gensim":       "gensim",
	"textblob":     "textblob",

	// Image Processing
	"imageio": "imageio",
	"imutils": "imutils",

	// Audio/Video
	"librosa":   "librosa",
	"soundfile": "soundfile",
	"pydub":     "pydub",
	"moviepy":   "moviepy",

	// CLI & System
	"click":    "click",
	"typer":    "typer",
	"rich":     "rich",
	"tqdm":     "tqdm",
	"colorama": "colorama",

	// Testing
	"pytest":     "pytest",
	"mock":       "mock",
	"faker":      "Faker",
	"hypothesis": "hypothesis",

	// Finance
	"yfinance":          "yfinance",
	"pandas_datareader": "pandas-datareader",
	"ta":                "ta",
	"backtrader":        "backtrader",

	// Misc
	"magic":     "python-magic",
	"serial":    "pyserial",
	"wx":        "wxPython",
	"pyautogui": "pyautogui",
	"pynput":    "pynput",
	"schedule":  "schedule",
	"arrow":     "arrow",
	"pendulum":  "pendulum",
}

// ResolvePackageName converts a Python module name to its pip package name.
// It handles submodule imports (e.g., "sklearn.ensemble" -> "scikit-learn")
// and falls back to the module name if no mapping exists.
func ResolvePackageName(moduleName string) string {
	// Handle submodule imports (e.g., "sklearn.ensemble" -> "sklearn")
	parts := strings.Split(moduleName, ".")
	rootModule := parts[0]

	// Check if root module has a mapping
	if pkg, ok := moduleToPackage[rootModule]; ok {
		return pkg
	}

	// Check full module path (for cases like "google.cloud")
	if len(parts) >= 2 {
		twoLevel := parts[0] + "." + parts[1]
		if pkg, ok := moduleToPackage[twoLevel]; ok {
			return pkg
		}
	}

	// Fallback: most packages match their module name
	return rootModule
}

// GetCommonPackages returns a list of commonly used packages that can be
// pre-installed to avoid runtime installation delays.
func GetCommonPackages() []string {
	return []string{
		"numpy",
		"pandas",
		"matplotlib",
		"scikit-learn",
		"requests",
		"beautifulsoup4",
		"seaborn",
		"scipy",
		"Pillow",
		"python-dateutil",
	}
}
