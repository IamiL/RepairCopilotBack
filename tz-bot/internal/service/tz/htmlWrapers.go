package tzservice

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
)

func WrapSubstring(s, sub, id string) (string, bool) {
	if sub == "" {
		return s, false
	}
	if !strings.Contains(s, sub) {
		return s, false
	}
	wrapped := `<span error-id="` + html.EscapeString(id) + `">` + sub + `</span>`
	return strings.ReplaceAll(s, sub, wrapped), true
}

// WrapApproxHTML ищет подстроку sub в текстовом содержимом htmlStr,
// позволяя подстроке "жить" частично внутри html-тегов (т.е. игнорируя теги при поиске).
// Если найдена, оборачивает соответствующий диапазон узлов/частей узлов в <span error-id="id">...</span>.
// Возвращает (результат, найдена_ли_подстрока, error).
func WrapSubstringApproxHTML(htmlStr, sub, id string) (string, bool, error) {
	if strings.TrimSpace(sub) == "" {
		return htmlStr, false, nil
	}

	// Парсим как фрагмент (htmlStr может быть кусочком, а не полным документом).
	container := &html.Node{Type: html.ElementNode, Data: "div"}
	frags, err := html.ParseFragment(strings.NewReader(htmlStr), container)
	if err != nil {
		return htmlStr, false, fmt.Errorf("parse html: %w", err)
	}
	for _, f := range frags {
		container.AppendChild(f)
	}

	// Собираем плоский текст и карту позиций -> текстовые узлы
	type seg struct {
		node       *html.Node // текстовый узел
		start, end int        // глобальные индексы [start, end)
	}
	var segs []seg
	var flat strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			txt := n.Data
			if txt != "" {
				s := flat.Len()
				flat.WriteString(txt)
				e := flat.Len()
				segs = append(segs, seg{node: n, start: s, end: e})
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(container)

	plain := flat.String()
	idx := strings.Index(plain, sub)
	if idx < 0 {
		return htmlStr, false, nil
	}
	matchStart := idx
	matchEnd := idx + len(sub)

	// Находим какие текстовые узлы покрывает диапазон
	firstIdx, lastIdx := -1, -1
	for i, s := range segs {
		if matchStart < s.end && matchEnd > s.start { // пересечение
			if firstIdx == -1 {
				firstIdx = i
			}
			lastIdx = i
		}
	}
	if firstIdx == -1 {
		// На всякий — не нашли отображение (не должно случиться).
		return htmlStr, false, nil
	}

	firstSeg := segs[firstIdx]
	lastSeg := segs[lastIdx]

	// Проверим, что все задействованные узлы имеют общего родителя
	parent := firstSeg.node.Parent
	for i := firstIdx; i <= lastIdx; i++ {
		if segs[i].node.Parent != parent {
			// Для простоты текущей реализации — откажемся.
			// Можно доработать до поиска LCA и сложного перемещения.
			return htmlStr, false, fmt.Errorf("match spans across different parents; not supported in this simple implementation")
		}
	}

	// Разбиваем крайние текстовые узлы на before/mid/after (если нужно)
	// Хелпер: вставить новый текстовый узел перед n (и вернуть его)
	insertTextBefore := func(n *html.Node, text string) *html.Node {
		if text == "" {
			return nil
		}
		t := &html.Node{Type: html.TextNode, Data: text}
		n.Parent.InsertBefore(t, n)
		return t
	}
	// Хелпер: вставить новый текстовый узел после n (и вернуть его)
	insertTextAfter := func(n *html.Node, text string) *html.Node {
		if text == "" {
			return nil
		}
		t := &html.Node{Type: html.TextNode, Data: text}
		if n.NextSibling != nil {
			n.Parent.InsertBefore(t, n.NextSibling)
		} else {
			n.Parent.AppendChild(t)
		}
		return t
	}

	// Локальные границы внутри первого узла
	firstLocalStart := max(0, matchStart-firstSeg.start)
	firstLocalEnd := min(len(firstSeg.node.Data), matchEnd-firstSeg.start)
	// В первом узле: before | mid | after (mid может быть пустым, если диапазон начинается не здесь)
	if firstIdx == lastIdx {
		// Весь матч в одном текстовом узле
		before := firstSeg.node.Data[:firstLocalStart]
		mid := firstSeg.node.Data[firstLocalStart:firstLocalEnd]
		after := firstSeg.node.Data[firstLocalEnd:]

		// Заменим исходный узел на before + <span>mid</span> + after
		ref := firstSeg.node
		if before != "" {
			insertTextBefore(ref, before)
		}
		span := &html.Node{Type: html.ElementNode, Data: "span"}
		span.Attr = []html.Attribute{{Key: "error-id", Val: id}}
		parent.InsertBefore(span, ref)
		if mid != "" {
			span.AppendChild(&html.Node{Type: html.TextNode, Data: mid})
		}
		if after != "" {
			insertTextAfter(ref, after)
		}
		parent.RemoveChild(ref)

		// Сериализуем
		var buf bytes.Buffer
		for n := container.FirstChild; n != nil; n = n.NextSibling {
			html.Render(&buf, n)
		}
		return buf.String(), true, nil
	}

	// Иначе — матч покрывает несколько узлов
	// Обработаем первый и последний узлы частично, середние — целиком

	// --- первый узел: split
	{
		before := firstSeg.node.Data[:firstLocalStart]
		mid := firstSeg.node.Data[firstLocalStart:]
		ref := firstSeg.node
		if before != "" {
			insertTextBefore(ref, before)
		}
		// Перезапишем текущий узел на mid (он станет началом диапазона)
		firstSeg.node.Data = mid
	}

	// --- последний узел: split
	lastLocalEnd := max(0, matchEnd-lastSeg.start)
	if lastLocalEnd < len(lastSeg.node.Data) {
		// Разрежем на mid | after
		mid := lastSeg.node.Data[:lastLocalEnd]
		after := lastSeg.node.Data[lastLocalEnd:]
		ref := lastSeg.node
		// Вставим after после ref
		insertTextAfter(ref, after)
		// Оставим в ref только mid
		lastSeg.node.Data = mid
	}

	// Теперь все задействованные узлы — это:
	// первый: firstSeg.node (начинается где надо)
	// дальше: все полные текстовые/элементные узлы между ними (если есть)
	// последний: lastSeg.node (заканчивается где надо)
	// Все они — соседние братья внутри одного parent.

	// Создаём span
	span := &html.Node{Type: html.ElementNode, Data: "span"}
	span.Attr = []html.Attribute{{Key: "error-id", Val: id}}

	// Вставим span перед первым узлом
	parent.InsertBefore(span, firstSeg.node)

	// Переместим в span всё от firstSeg.node до lastSeg.node (включительно),
	// включая не только текстовые, но и любые элементы между ними.
	cur := firstSeg.node
	for {
		next := cur.NextSibling // запомним, потому что cur переместим
		span.AppendChild(cur)
		if cur == lastSeg.node {
			break
		}
		cur = next
	}

	// Сериализуем фрагмент обратно
	var buf bytes.Buffer
	for n := container.FirstChild; n != nil; n = n.NextSibling {
		html.Render(&buf, n)
	}
	return buf.String(), true, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
