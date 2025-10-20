"""
Надёжная вставка комментариев (Review) в .docx:
- создаём/обновляем word/comments.xml (w:comment c w:author, w:date, w:initials)
- добавляем связь в word/_rels/document.xml.rels (если её нет)
- ДОБАВЛЯЕМ Override в [Content_Types].xml для /word/comments.xml (иначе Word может «чинить» файл и терять текст)
- ставим w:commentRangeStart/End + w:commentReference вокруг целевого текста
- текст комментария — по строкам, каждый в отдельном абзаце; w:t с xml:space="preserve"
"""

from io import BytesIO
from typing import List, Dict
from zipfile import ZipFile, ZIP_DEFLATED
from datetime import datetime

from lxml import etree

W_NS = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
REL_NS = "http://schemas.openxmlformats.org/package/2006/relationships"
CT_NS = "http://schemas.openxmlformats.org/package/2006/content-types"
XML_NS = "http://www.w3.org/XML/1998/namespace"

def _ns(tag: str) -> str:
    return f"{{{W_NS}}}{tag}"

def _rel(tag: str) -> str:
    return f"{{{REL_NS}}}{tag}"

def _ct(tag: str) -> str:
    return f"{{{CT_NS}}}{tag}"

def _set_space_preserve(t_el):
    t_el.set(f"{{{XML_NS}}}space", "preserve")

def _append_comment_paragraph(parent, text_line: str):
    p = etree.SubElement(parent, _ns("p"))
    r = etree.SubElement(p, _ns("r"))
    t = etree.SubElement(r, _ns("t"))
    _set_space_preserve(t)
    t.text = text_line


def _ensure_content_types_override(ct_xml):
    """
    Гарантирует, что в [Content_Types].xml есть Override для /word/comments.xml
    """
    wanted_name = "/word/comments.xml"
    wanted_type = "application/vnd.openxmlformats-officedocument.wordprocessingml.comments+xml"

    # Есть ли уже Override для comments.xml?
    for ov in ct_xml.findall(_ct("Override")):
        if ov.get("PartName") == wanted_name:
            return  # уже есть

    # Добавим в конец
    ov = etree.SubElement(ct_xml, _ct("Override"))
    ov.set("PartName", wanted_name)
    ov.set("ContentType", wanted_type)


def inject_comments(docx_bytes: bytes, anchors: List[Dict[str, str]]) -> bytes:
    """
    anchors: список словарей:
      - 'text': точный текст якоря (обычно what_is_incorrect)
      - 'comment_lines': список строк, каждая — отдельный абзац в комментарии
    Возвращает новые байты .docx со вставленными комментариями.
    """
    in_mem = BytesIO(docx_bytes)
    out_mem = BytesIO()

    with ZipFile(in_mem, "r") as zin, ZipFile(out_mem, "w", compression=ZIP_DEFLATED) as zout:
        names = set(zin.namelist())

        doc_path = "word/document.xml"
        rels_path = "word/_rels/document.xml.rels"
        comments_path = "word/comments.xml"
        ct_path = "[Content_Types].xml"

        # читаем XML-части
        doc_xml = etree.fromstring(zin.read(doc_path))
        rels_xml = etree.fromstring(zin.read(rels_path)) if rels_path in names else etree.Element(_rel("Relationships"), nsmap={None: REL_NS})
        comments_xml = etree.fromstring(zin.read(comments_path)) if comments_path in names else etree.Element(_ns("comments"), nsmap={"w": W_NS})
        ct_xml = etree.fromstring(zin.read(ct_path)) if ct_path in names else etree.Element(_ct("Types"), nsmap={None: CT_NS})

        # 1) comments.xml — наполняем
        existing_ids = [int(c.get(_ns("id"))) for c in comments_xml.findall(_ns("comment"))] or [-1]
        next_id = max(existing_ids) + 1
        comment_ids = []

        ts = datetime.utcnow().replace(microsecond=0).isoformat() + "Z"

        for a in anchors:
            c_el = etree.Element(_ns("comment"), nsmap={"w": W_NS})
            c_el.set(_ns("id"), str(next_id))
            c_el.set(_ns("author"), "Auto")
            c_el.set(_ns("initials"), "AI")
            c_el.set(_ns("date"), ts)

            lines = a.get("comment_lines") or []
            if not lines:
                # fallback для старых вызовов с ключом "comment"
                fallback = a.get("comment") or ""
                lines = [fallback]

            for line in lines:
                _append_comment_paragraph(c_el, line)

            comments_xml.append(c_el)
            comment_ids.append(next_id)
            next_id += 1

        # 2) document.xml.rels — ensure rel to comments.xml
        comments_rel_type = "http://schemas.openxmlformats.org/officeDocument/2006/relationships/comments"
        existing_rels = rels_xml.findall(_rel("Relationship"))
        if not any(r.get("Type") == comments_rel_type for r in existing_rels):
            used_ids = []
            for r in existing_rels:
                rid = r.get("Id", "")
                if rid.startswith("rId"):
                    try:
                        used_ids.append(int(rid[3:]))
                    except Exception:
                        pass
            new_rid = f"rId{(max(used_ids) + 1) if used_ids else 1}"
            rel_el = etree.SubElement(rels_xml, _rel("Relationship"))
            rel_el.set("Id", new_rid)
            rel_el.set("Type", comments_rel_type)
            rel_el.set("Target", "comments.xml")

        # 3) [Content_Types].xml — ensure Override for comments.xml
        _ensure_content_types_override(ct_xml)

        # 4) document.xml — проставляем якоря
        all_text_nodes = doc_xml.findall(".//" + _ns("t"))

        def make_ref_run(cid: int):
            r = etree.Element(_ns("r"))
            rPr = etree.SubElement(r, _ns("rPr"))
            cref = etree.SubElement(r, _ns("commentReference"))
            cref.set(_ns("id"), str(cid))
            return r

        idx = 0
        for t_node in all_text_nodes:
            if idx >= len(anchors):
                break
            anchor_text = (anchors[idx]["text"] or "").strip()
            current_text = (t_node.text or "").strip()
            if current_text == anchor_text:
                r_node = t_node.getparent()

