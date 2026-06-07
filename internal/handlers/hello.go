package handlers

import (
	_ "embed"
	"net/http"

	"go.uber.org/zap"
)

//go:embed logo.png
var logoBytes []byte

type HelloHandler struct {
	logger *zap.Logger
}

func NewHelloHandler(logger *zap.Logger) *HelloHandler {
	return &HelloHandler{logger: logger}
}

// Hello handles GET /hello
// @Summary      Hello Endpoint Web Page
// @Description  Returns a simple web page with the centered logo and "hecho en Colombia" footer.
// @Tags         system
// @Produce      html
// @Success      200      {string}  string "HTML page"
// @Router       /hello [get]
func (h *HelloHandler) Hello(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	html := `<!DOCTYPE html>
<html lang="es">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Ayurami</title>
    <style>
        body {
            margin: 0;
            padding: 0;
            display: flex;
            flex-direction: column;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            background-color: #f7f9fa;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
        }
        .container {
            text-align: center;
            flex: 1;
            display: flex;
            justify-content: center;
            align-items: center;
            padding: 20px;
        }
        img {
            max-width: 100%;
            max-height: 70vh;
            height: auto;
            border-radius: 20px;
            box-shadow: 0 10px 30px rgba(0, 0, 0, 0.05);
            transition: transform 0.3s ease;
        }
        img:hover {
            transform: scale(1.02);
        }
        footer {
            width: 100%;
            padding: 20px 0;
            text-align: center;
            color: #7f8c8d;
            font-size: 0.9rem;
            font-weight: 500;
            letter-spacing: 0.5px;
            border-top: 1px solid #eaeaea;
            background-color: #ffffff;
        }
    </style>
</head>
<body>
    <div class="container">
        <img src="/hello/logo.png" alt="Ayurami logo">
    </div>
    <footer>
        Hecho en Colombia
    </footer>
</body>
</html>`

	if _, err := w.Write([]byte(html)); err != nil {
		h.logger.Error("failed to write hello html response", zap.Error(err))
	}
}

// Logo handles GET /hello/logo.png
// @Summary      Logo image
// @Description  Returns the embedded Ayurami logo image bytes.
// @Tags         system
// @Produce      image/png
// @Success      200      {string}  string "logo image bytes"
// @Router       /hello/logo.png [get]
func (h *HelloHandler) Logo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(logoBytes); err != nil {
		h.logger.Error("failed to write hello logo bytes", zap.Error(err))
	}
}
