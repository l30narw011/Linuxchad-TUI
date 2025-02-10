package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Post representa una publicación en el foro
type Post struct {
	Cooked    string `json:"cooked"`   // Contenido en HTML
	Username  string `json:"username"` // Nombre del usuario
	Reactions []struct {
		Name  string `json:"id"`
		Count int    `json:"count"`
	} `json:"reactions"`
}

// Estructuras de respuesta JSON
type (
	TopicDetailResponse struct {
		PostStream struct {
			Posts []Post `json:"posts"`
		} `json:"post_stream"`
	}
	Topic struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	}
	LatestTopicsResponse struct {
		TopicList struct {
			Topics []Topic `json:"topics"`
		} `json:"topic_list"`
	}
)

// Fetch JSON desde una URL y decodificar en `target`
func fetchJSON(url string, target interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(target)
}

// Obtener los temas recientes
func getLatestTopics() ([]Topic, error) {
	var data LatestTopicsResponse
	return data.TopicList.Topics, fetchJSON("https://foro.linuxchad.org/latest.json", &data)
}

// Obtener respuestas de un tema
func getTopicReplies(topicID int) ([]string, error) {
	var data TopicDetailResponse
	if err := fetchJSON(fmt.Sprintf("https://foro.linuxchad.org/t/%d.json", topicID), &data); err != nil {
		return nil, err
	}
	var replies []string
	for _, post := range data.PostStream.Posts {
		replies = append(replies, fmt.Sprintf("👤 %s\n%s\n⭐ %s\n📷 %s", post.Username, stripHTML(post.Cooked), formatReactions(post.Reactions), extractImages(post.Cooked)))
	}
	return replies, nil
}

// Eliminar etiquetas HTML y mejorar formato
func stripHTML(html string) string {
	replacer := strings.NewReplacer("<p>", "", "</p>", "\n", "<br>", "\n", "&amp;", "&", "&lt;", "<", "&gt;", ">")
	text := replacer.Replace(html)
	return regexp.MustCompile(`<[^>]*>`).ReplaceAllString(text, "")
}

// Formatear reacciones a texto
func formatReactions(reactions []struct {
	Name  string `json:"id"`
	Count int    `json:"count"`
}) string {
	if len(reactions) == 0 {
		return "Sin reacciones."
	}
	var out []string
	for _, r := range reactions {
		out = append(out, fmt.Sprintf("%s: %d", r.Name, r.Count))
	}
	return strings.Join(out, ", ")
}

// Extraer enlaces de imágenes
func extractImages(html string) string {
	re := regexp.MustCompile(`<img[^>]*src="([^"]+)"[^>]*>`)
	matches := re.FindAllStringSubmatch(html, -1)
	var images []string
	for _, match := range matches {
		images = append(images, match[1])
	}
	if len(images) == 0 {
		return "Sin imágenes."
	}
	return strings.Join(images, ", ")
}

func main() {
	app := tview.NewApplication()
	list := tview.NewList()
	repliesView := tview.NewTextView().SetDynamicColors(true)

	// Cargar temas
	topics, err := getLatestTopics()
	if err != nil {
		log.Fatalf("Error al obtener los temas: %v", err)
	}

	// Agregar temas a la lista
	for _, topic := range topics {
		topicID := topic.ID
		list.AddItem(topic.Title, "", 0, func() {
			replies, err := getTopicReplies(topicID)
			if err != nil {
				repliesView.SetText("Error al cargar respuestas.")
			} else {
				repliesView.SetText(strings.Join(replies, "\n\n━━━━━━━━━━━━━━━━━━━━\n\n"))
			}
			app.SetRoot(repliesView, true).SetFocus(repliesView)
		})
	}

	// Permitir volver atrás con ESC
	repliesView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			app.SetRoot(list, true).SetFocus(list)
		}
		return event
	})

	// Mostrar la lista
	if err := app.SetRoot(list, true).Run(); err != nil {
		log.Fatalf("Error en la aplicación: %v", err)
	}
}
