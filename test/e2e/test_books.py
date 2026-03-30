"""E2E tests for book library endpoints."""

import io
import os
import zipfile

import httpx
import pytest

REAL_EPUB = os.path.join(os.path.dirname(__file__), "..", "97thingseveryapplicationsecurityprofessionalshouldknow.epub")


def make_minimal_epub(title: str = "Test Book", author: str = "Test Author") -> bytes:
    """Creates a minimal valid EPUB file in-memory for testing."""
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED) as zf:
        zf.writestr("mimetype", "application/epub+zip")
        zf.writestr(
            "META-INF/container.xml",
            '<?xml version="1.0"?>'
            '<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">'
            "<rootfiles>"
            '<rootfile full-path="content.opf" media-type="application/oebps-package+xml"/>'
            "</rootfiles></container>",
        )
        zf.writestr(
            "content.opf",
            f'<?xml version="1.0" encoding="UTF-8"?>'
            f'<package xmlns="http://www.idpf.org/2007/opf" version="2.0">'
            f'<metadata xmlns:dc="http://purl.org/dc/elements/1.1/">'
            f"<dc:title>{title}</dc:title>"
            f"<dc:creator>{author}</dc:creator>"
            f"<dc:language>en</dc:language>"
            f"</metadata>"
            f'<manifest><item id="ncx" href="toc.ncx" media-type="application/x-dtbncx+xml"/></manifest>'
            f'<spine toc="ncx"></spine>'
            f"</package>",
        )
    return buf.getvalue()


def test_list_books(auth_client: httpx.Client):
    r = auth_client.get("/api/v1/books")
    assert r.status_code == 200
    data = r.json()
    assert "books" in data
    assert isinstance(data["books"], list)
    assert "total" in data


def test_upload_epub(auth_client: httpx.Client):
    epub_bytes = make_minimal_epub("My Test Book", "Jane Doe")
    r = auth_client.post(
        "/api/v1/books",
        files={"file": ("test.epub", epub_bytes, "application/epub+zip")},
    )
    assert r.status_code == 201
    book = r.json()["book"]
    assert book["title"] == "My Test Book"
    assert book["author"] == "Jane Doe"
    assert book["file_type"] == "epub"
    assert "id" in book
    return book


def test_get_book(auth_client: httpx.Client):
    # Upload first
    epub = make_minimal_epub("Get Me", "Author")
    upload_r = auth_client.post(
        "/api/v1/books",
        files={"file": ("get_me.epub", epub, "application/epub+zip")},
    )
    assert upload_r.status_code == 201
    book_id = upload_r.json()["book"]["id"]

    r = auth_client.get(f"/api/v1/books/{book_id}")
    assert r.status_code == 200
    assert r.json()["book"]["id"] == book_id


def test_update_book(auth_client: httpx.Client):
    epub = make_minimal_epub("Old Title", "Old Author")
    upload_r = auth_client.post(
        "/api/v1/books",
        files={"file": ("update_me.epub", epub, "application/epub+zip")},
    )
    book_id = upload_r.json()["book"]["id"]

    r = auth_client.put(f"/api/v1/books/{book_id}", json={"title": "New Title"})
    assert r.status_code == 200
    assert r.json()["book"]["title"] == "New Title"


def test_delete_book(auth_client: httpx.Client):
    epub = make_minimal_epub("Delete Me")
    upload_r = auth_client.post(
        "/api/v1/books",
        files={"file": ("delete_me.epub", epub, "application/epub+zip")},
    )
    book_id = upload_r.json()["book"]["id"]

    r = auth_client.delete(f"/api/v1/books/{book_id}")
    assert r.status_code == 200

    r2 = auth_client.get(f"/api/v1/books/{book_id}")
    assert r2.status_code == 404


@pytest.mark.skipif(not os.path.exists(REAL_EPUB), reason="real epub fixture not present")
def test_upload_real_epub(auth_client: httpx.Client):
    with open(REAL_EPUB, "rb") as f:
        r = auth_client.post(
            "/api/v1/books",
            files={"file": (os.path.basename(REAL_EPUB), f, "application/epub+zip")},
        )
    assert r.status_code == 201, r.text
    book = r.json()["book"]
    assert book["file_type"] == "epub"
    assert book["title"] != ""


def test_upload_unsupported_type(auth_client: httpx.Client):
    r = auth_client.post(
        "/api/v1/books",
        files={"file": ("notes.txt", b"hello", "text/plain")},
    )
    assert r.status_code == 400


def test_save_and_get_progress(auth_client: httpx.Client):
    epub = make_minimal_epub("Progress Book")
    upload_r = auth_client.post(
        "/api/v1/books",
        files={"file": ("progress.epub", epub, "application/epub+zip")},
    )
    book_id = upload_r.json()["book"]["id"]

    # Save progress
    r = auth_client.put(
        f"/api/v1/books/{book_id}/progress",
        json={"cfi": "epubcfi(/6/4!/4/2/2:0)", "percentage": 0.25},
    )
    assert r.status_code == 200
    progress = r.json()["progress"]
    assert progress["percentage"] == pytest.approx(0.25)

    # Retrieve progress
    r2 = auth_client.get(f"/api/v1/books/{book_id}/progress")
    assert r2.status_code == 200
    assert r2.json()["progress"]["percentage"] == pytest.approx(0.25)


def test_book_content_served(auth_client: httpx.Client):
    epub = make_minimal_epub("Serve Me")
    upload_r = auth_client.post(
        "/api/v1/books",
        files={"file": ("serve_me.epub", epub, "application/epub+zip")},
    )
    book_id = upload_r.json()["book"]["id"]

    r = auth_client.get(f"/api/v1/books/{book_id}/content")
    assert r.status_code == 200
    assert "epub" in r.headers.get("content-type", "")
