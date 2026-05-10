import pyautogui
import requests
import time
import re
import cv2
import numpy as np
import easyocr
from concurrent.futures import ThreadPoolExecutor

# ── Init sekali saja ──
reader = easyocr.Reader(['en'], gpu=False, verbose=False)
executor = ThreadPoolExecutor(max_workers=1)  # thread khusus untuk HTTP

REGION_HURUF = (1390, 880, 620, 60)
LAST_WORD = ""
INTERVAL = 0.6  # detik antar capture (naikkan jika masih berat)

def preprocess(img_cv):
    """Crop ROI kotak putih terbesar."""
    gray = cv2.cvtColor(img_cv, cv2.COLOR_BGR2GRAY)
    _, mask = cv2.threshold(gray, 200, 255, cv2.THRESH_BINARY)
    contours, _ = cv2.findContours(mask, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE)

    if not contours:
        return None

    # Ambil kotak terbesar langsung tanpa list comprehension
    biggest = max(
        (cv2.boundingRect(c) for c in contours if cv2.boundingRect(c)[2] * cv2.boundingRect(c)[3] > 1000),
        key=lambda b: b[2] * b[3],
        default=None
    )
    if biggest is None:
        return None

    x0, y0, w0, h0 = biggest
    m = 6
    roi = img_cv[max(0, y0+m):y0+h0-m, max(0, x0+m):x0+w0-m]
    return roi if roi.size > 0 else None

def capture_and_read():
    # Screenshot langsung ke numpy — skip PIL save ke disk
    screenshot = pyautogui.screenshot(region=REGION_HURUF)
    img_cv = cv2.cvtColor(np.array(screenshot), cv2.COLOR_RGB2BGR)

    roi = preprocess(img_cv)
    if roi is None:
        return ""

    # OCR — matikan detail supaya lebih cepat
    results = reader.readtext(
        roi,
        allowlist='ABCDEFGHIJKLMNOPQRSTUVWXYZ',
        detail=0,          # ← hanya return string, skip koordinat
        paragraph=True     # ← gabung otomatis jadi 1 string
    )

    if not results:
        return ""

    raw = re.sub(r'[^A-Z]', '', "".join(results).upper())
    print(f"  OCR: '{raw}'")
    return raw

def send_to_web(word):
    """Kirim HTTP di background thread — tidak block loop utama."""
    def _send():
        try:
            requests.get(
                f"http://localhost:8000/auto-input?q={word.lower()}",
                timeout=1
            )
        except Exception as e:
            print(f"  Web error: {e}")
    executor.submit(_send)

# ── Main loop ──
print("OCR berjalan... tekan Ctrl+C untuk stop\n")

# Warmup EasyOCR sekali biar tidak lambat di deteksi pertama
_blank = np.ones((40, 200, 3), dtype=np.uint8) * 255
reader.readtext(_blank, detail=0)
print("Warmup selesai, mulai deteksi...\n")

while True:
    try:
        word = capture_and_read()
        if word and word != LAST_WORD and len(word) >= 1:
            print(f"✅ Detected: '{word}'")
            send_to_web(word)
            LAST_WORD = word
    except Exception as e:
        print(f"  Loop error: {e}")

    time.sleep(INTERVAL)