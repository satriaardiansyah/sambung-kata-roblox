import pyautogui
import requests
import time
import re
import cv2
import numpy as np
import easyocr
import hashlib

reader = easyocr.Reader(['en'], gpu=False)

REGION_HURUF = (1400, 875, 720, 70)  # lebih lebar & tinggi dari sebelumnya
LAST_WORD = ""
LAST_HASH = ""
FAIL_COUNT = 0  # hitung berapa kali gagal deteksi berturut-turut

def img_hash(gray):
    return hashlib.md5(gray.tobytes()).hexdigest()

def try_ocr(roi, label=""):
    """Coba OCR dengan berbagai ukuran resize, return hasil terbaik."""
    for scale in [1, 2, 3]:
        h, w = roi.shape[:2]
        resized = cv2.resize(roi, (w*scale, h*scale), interpolation=cv2.INTER_CUBIC)
        results = reader.readtext(
            resized,
            allowlist='ABCDEFGHIJKLMNOPQRSTUVWXYZ',
            detail=1,
            batch_size=1,
            workers=0,
        )
        if results:
            results.sort(key=lambda r: r[0][0][0])
            raw = "".join(r[1] for r in results).upper()
            raw = re.sub(r'[^A-Z]', '', raw)
            if raw:
                print(f"  OCR ({label}, scale={scale}x): '{raw}'")
                return raw
    return ""

def capture_and_read():
    global LAST_HASH, FAIL_COUNT

    screenshot = pyautogui.screenshot(region=REGION_HURUF)
    img_cv = cv2.cvtColor(np.array(screenshot), cv2.COLOR_RGB2BGR)
    gray = cv2.cvtColor(img_cv, cv2.COLOR_BGR2GRAY)

    h = img_hash(gray)
    if h == LAST_HASH:
        return None
    LAST_HASH = h

    screenshot.save("debug_region.png")

    # Step 1: Temukan kotak putih
    _, white_mask = cv2.threshold(gray, 200, 255, cv2.THRESH_BINARY)
    cv2.imwrite("debug_white_mask.png", white_mask)
    contours, _ = cv2.findContours(white_mask, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE)

    boxes = []
    for cnt in contours:
        x, y, w, h = cv2.boundingRect(cnt)
        if w * h > 1000 and 0.3 < (w/h) < 5.0 and h > 15:
            boxes.append((x, y, w, h))

    if not boxes:
        FAIL_COUNT += 1
        print(f"  ⚠️ [FAIL #{FAIL_COUNT}] Kotak putih tidak ditemukan — cek debug_region.png & debug_white_mask.png")
        return ""

    boxes.sort(key=lambda b: b[0])
    x0 = min(b[0] for b in boxes)
    y0 = min(b[1] for b in boxes)
    x1 = max(b[0]+b[2] for b in boxes)
    y1 = max(b[1]+b[3] for b in boxes)

    roi_w = x1 - x0
    roi_h = y1 - y0
    print(f"  📦 Box ditemukan: {len(boxes)} kotak, ROI size: {roi_w}x{roi_h}px")

    # Debug: gambar kotak yang terdeteksi
    debug_box = img_cv.copy()
    for (bx, by, bw, bh) in boxes:
        cv2.rectangle(debug_box, (bx, by), (bx+bw, by+bh), (0, 255, 0), 2)
    cv2.imwrite("debug_boxes.png", debug_box)

    # Step 2: Crop dalam kotak
    margin = 6
    roi = img_cv[
        max(0, y0+margin):y1-margin,
        max(0, x0+margin):x1-margin
    ]
    cv2.imwrite("debug_roi.png", roi)

    if roi.size == 0:
        FAIL_COUNT += 1
        print(f"  ⚠️ [FAIL #{FAIL_COUNT}] ROI kosong setelah crop — margin terlalu besar?")
        return ""

    # Step 3: OCR dengan fallback multi-scale
    raw = try_ocr(roi, label="roi")

    if not raw:
        FAIL_COUNT += 1
        print(f"  ⚠️ [FAIL #{FAIL_COUNT}] EasyOCR gagal baca — cek debug_roi.png")
        print(f"     ROI size: {roi.shape[1]}x{roi.shape[0]}px")
        # Simpan ROI yang gagal untuk analisis
        cv2.imwrite(f"debug_fail_{FAIL_COUNT}.png", roi)
        return ""

    FAIL_COUNT = 0  # reset jika berhasil
    return raw

def send_to_web(word):
    try:
        requests.get(f"http://localhost:8000/auto-input?q={word.lower()}", timeout=1)
        print(f"  Sent: {word}")
    except Exception as e:
        print(f"  Web error: {e}")

print("OCR berjalan... tekan Ctrl+C untuk stop\n")

while True:
    word = capture_and_read()
    if word is None:
        time.sleep(0.3)
        continue
    if word and word != LAST_WORD and len(word) >= 1:
        print(f"✅ Detected: '{word}'")
        send_to_web(word)
        LAST_WORD = word
    time.sleep(0.5)