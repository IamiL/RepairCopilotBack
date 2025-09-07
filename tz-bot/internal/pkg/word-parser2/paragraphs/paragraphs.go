package paragraphsproc

import (
	"bytes"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

const (
	attrArticleIdx      = "data-x-extract-article-index"
	attrChildIdx        = "data-x-extract-child-index"
	placeholderPrefix   = "X-EXTRACT-PLACEHOLDER:"
	wrapperRootID       = "__x_root__"
	wrapperParagraphsID = "__x_paras__"
)

// --------------------------- Публичные функции ---------------------------

func ExtractParagraphs(input string) (htmlWithPlaceholders string, paragraphs string) {
	doc := mustParseFullDoc(wrapWithRootDiv(input, wrapperRootID))
	root := mustFindByID(doc, wrapperRootID)
	if root == nil {
		// На всякий случай — возвращаем исходник без изменений
		return input, ""
	}

	var extracted []*html.Node
	articleIndex := 0

	walk(root, func(n *html.Node) {
		if isElement(n, "article") {
			extractFromArticle(n, articleIndex, &extracted)
			articleIndex++
		}
	})

	return renderChildren(root), renderNodes(extracted)
}

func InsertParagraphs(htmlWithPlaceholder, paragraphs string) string {
	// Парсим основной HTML с искусственным корнем
	doc := mustParseFullDoc(wrapWithRootDiv(htmlWithPlaceholder, wrapperRootID))
	root := mustFindByID(doc, wrapperRootID)
	if root == nil {
		return htmlWithPlaceholder
	}

	// Парсим «плоские» параграфы (как полноценный документ с отдельным корнем)
	parasDoc := mustParseFullDoc(wrapWithRootDiv(paragraphs, wrapperParagraphsID))
	parasRoot := mustFindByID(parasDoc, wrapperParagraphsID)
	if parasRoot == nil {
		return htmlWithPlaceholder
	}

	// Сгруппируем элементы: articleIdx -> [] (childIdx, *Node)
	type pair struct {
		idx  int
		node *html.Node
	}
	group := map[int][]pair{}

	for c := parasRoot.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		aStr, okA := getAttr(c, attrArticleIdx)
		iStr, okI := getAttr(c, attrChildIdx)
		if !okA || !okI {
			continue
		}
		aIdx, errA := strconv.Atoi(aStr)
		iIdx, errI := strconv.Atoi(iStr)
		if errA != nil || errI != nil {
			continue
		}
		// Уберём служебные атрибуты перед вставкой
		removeAttr(c, attrArticleIdx)
		removeAttr(c, attrChildIdx)

		group[aIdx] = append(group[aIdx], pair{idx: iIdx, node: c})
	}

	for k := range group {
		sort.Slice(group[k], func(i, j int) bool { return group[k][i].idx < group[k][j].idx })
	}

	// Найдём плейсхолдеры-комментарии и вставим элементы обратно
	walk(root, func(n *html.Node) {
		if n.Type == html.CommentNode && strings.HasPrefix(n.Data, placeholderPrefix) {
			parent := n.Parent
			if parent == nil {
				return
			}
			// Извлечь индекс article из комментария
			idxStr := strings.TrimPrefix(n.Data, placeholderPrefix)
			aIdx, err := strconv.Atoi(strings.TrimSpace(idxStr))
			if err != nil {
				return
			}
			// Удаляем плейсхолдер
			removeNode(n)
			// Вставляем элементы этой группы (если есть)
			for _, p := range group[aIdx] {
				if p.node.Parent != nil {
					removeNode(p.node)
				}
				parent.AppendChild(p.node)
			}
		}
	})

	return renderChildren(root)
}

// --------------------------- Внутренняя логика ---------------------------

func extractFromArticle(article *html.Node, articleIdx int, extracted *[]*html.Node) {
	// Собираем прямых дочерних ЭЛЕМЕНТОВ
	var children []*html.Node
	for c := article.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode {
			children = append(children, c)
		}
	}

	// Клонируем каждый элемент и добавляем служебные атрибуты
	for i, child := range children {
		cp := cloneDeep(child)
		setOrReplaceAttr(cp, attrArticleIdx, strconv.Itoa(articleIdx))
		setOrReplaceAttr(cp, attrChildIdx, strconv.Itoa(i))
		*extracted = append(*extracted, cp)
	}

	// Очищаем article полностью и ставим плейсхолдер-комментарий
	clearChildren(article)
	ph := &html.Node{Type: html.CommentNode, Data: placeholderPrefix + strconv.Itoa(articleIdx)}
	article.AppendChild(ph)
}

func wrapWithRootDiv(inner, id string) string {
	// Полноценная страница: это стабилизирует парсинг во всех окружениях
	return "<!DOCTYPE html><html><head></head><body><div id=\"" + id + "\">" + inner + "</div></body></html>"
}

func mustParseFullDoc(s string) *html.Node {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		// Возвращать nil не хочется — отдадим пустой документ с корнем <html>
		empty, _ := html.Parse(strings.NewReader("<!DOCTYPE html><html><head></head><body></body></html>"))
		return empty
	}
	return doc
}

func mustFindByID(n *html.Node, id string) *html.Node {
	var out *html.Node
	walk(n, func(x *html.Node) {
		if out != nil {
			return
		}
		if x.Type == html.ElementNode {
			if v, ok := getAttr(x, "id"); ok && v == id {
				out = x
			}
		}
	})
	return out
}

// --------------------------- Утилиты для DOM ---------------------------

func isElement(n *html.Node, name string) bool {
	return n != nil && n.Type == html.ElementNode && strings.EqualFold(n.Data, name)
}

func walk(n *html.Node, visit func(*html.Node)) {
	if n == nil {
		return
	}
	visit(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, visit)
	}
}

func cloneDeep(n *html.Node) *html.Node {
	if n == nil {
		return nil
	}
	cp := &html.Node{
		Type:      n.Type,
		Data:      n.Data,
		Namespace: n.Namespace,
	}
	if len(n.Attr) > 0 {
		cp.Attr = append(cp.Attr, n.Attr...)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		cp.AppendChild(cloneDeep(c))
	}
	return cp
}

func clearChildren(n *html.Node) {
	for c := n.FirstChild; c != nil; {
		next := c.NextSibling
		removeNode(c)
		c = next
	}
}

func removeNode(n *html.Node) {
	if n == nil || n.Parent == nil {
		return
	}
	p := n.Parent
	if n.PrevSibling != nil {
		n.PrevSibling.NextSibling = n.NextSibling
	} else {
		p.FirstChild = n.NextSibling
	}
	if n.NextSibling != nil {
		n.NextSibling.PrevSibling = n.PrevSibling
	} else {
		p.LastChild = n.PrevSibling
	}
	n.Parent = nil
	n.PrevSibling = nil
	n.NextSibling = nil
}

func renderChildren(n *html.Node) string {
	var buf bytes.Buffer
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		_ = html.Render(&buf, c)
	}
	return buf.String()
}

func renderNodes(list []*html.Node) string {
	var buf bytes.Buffer
	for _, n := range list {
		_ = html.Render(&buf, n)
	}
	return buf.String()
}

func getAttr(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

func setOrReplaceAttr(n *html.Node, key, val string) {
	for i := range n.Attr {
		if n.Attr[i].Key == key {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
}

func removeAttr(n *html.Node, key string) {
	i := 0
	for i < len(n.Attr) {
		if n.Attr[i].Key == key {
			n.Attr = append(n.Attr[:i], n.Attr[i+1:]...)
		} else {
			i++
		}
	}
}

// --------------------------- Мини-демо ---------------------------
//
// func main() {
// 	src := `<div>
//         <section>
//            <article class="a">
//               <h2 id="t">Title</h2>
//               <p role="x">Hello <b>world</b></p>
//               <div data-x="1"><span>inner</span></div>
//            </article>
//            <aside>side</aside>
//         </section>
//         <section>
//            <article data-q="z">
//               <p>Another</p><p>One</p>
//            </article>
//         </section>
//     </div>`
//
// 	withPH, flat := ExtractParagraphs(src)
// 	fmt.Println("WITH PLACEHOLDERS:\n", withPH)
// 	fmt.Println("\nPARAGRAPHS:\n", flat)
// 	restored := InsertParagraphs(withPH, flat)
// 	fmt.Println("\nRESTORED:\n", restored)
// }
//
// Ожидаемо:
//  - WITH PLACEHOLDERS: в каждом <article> только <!--X-EXTRACT-PLACEHOLDER:N-->
//  - PARAGRAPHS: последовательность <h2>…</h2><p …>…</p><div …>…</div><p>Another</p><p>One</p>
//  - RESTORED: совпадает по структуре и атрибутам с исходным (небольшие отличия в пробелах допустимы)
// ----------------------------------------------------------------
