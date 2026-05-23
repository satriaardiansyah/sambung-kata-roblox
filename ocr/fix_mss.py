import warnings
warnings.filterwarnings("ignore")

import mss
import requests
import time
import re
import cv2
import numpy as np
import easyocr
import hashlib
from concurrent.futures import ThreadPoolExecutor

reader   = easyocr.Reader(['en'], gpu=False, verbose=False)
executor = ThreadPoolExecutor(max_workers=1)

REGION_HURUF = (1390, 867, 238, 72)
LAST_WORD    = ""
INTERVAL     = 0.4
last_hash    = ""

# ✅ Buat MSS sekali, simpan sebagai global
_sct = mss.MSS()

def get_sct():
    """Kembalikan sct global, recreate kalau rusak."""
    global _sct
    try:
        # Test apakah masih hidup
        _sct.monitors
        return _sct
    except Exception:
        print("  ♻️ Recreate MSS context...")
        try:
            _sct.close()
        except Exception:
            pass
        _sct = mss.MSS()
        return _sct

def image_hash(img_cv):
    small = cv2.resize(img_cv, (32, 16))
    return hashlib.md5(small.tobytes()).hexdigest()

def preprocess(img_cv):
    gray = cv2.cvtColor(img_cv, cv2.COLOR_BGR2GRAY)
    _, mask = cv2.threshold(gray, 200, 255, cv2.THRESH_BINARY)
    contours, _ = cv2.findContours(mask, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE)
    if not contours:
        return None
    biggest = max(
        (cv2.boundingRect(c) for c in contours
         if cv2.boundingRect(c)[2] * cv2.boundingRect(c)[3] > 1000),
        key=lambda b: b[2] * b[3],
        default=None
    )
    if biggest is None:
        return None
    x0, y0, w0, h0 = biggest
    m   = 6
    roi = img_cv[max(0, y0+m):y0+h0-m, max(0, x0+m):x0+w0-m]
    return roi if roi.size > 0 else None

def capture_and_read():
    global last_hash

    monitor = {
        "left":   REGION_HURUF[0],
        "top":    REGION_HURUF[1],
        "width":  REGION_HURUF[2],
        "height": REGION_HURUF[3]
    }

    # ✅ Pakai sct yang sama, tidak buka/tutup tiap capture
    sct = get_sct()
    screenshot = sct.grab(monitor)
    img_cv = cv2.cvtColor(np.array(screenshot), cv2.COLOR_BGRA2BGR)

    # Skip OCR kalau gambar tidak berubah
    h = image_hash(img_cv)
    if h == last_hash:
        return ""
    last_hash = h

    roi = preprocess(img_cv)
    if roi is None:
        return ""

    results = reader.readtext(
        roi,
        allowlist='ABCDEFGHIJKLMNOPQRSTUVWXYZ',
        detail=0,
        paragraph=True
    )
    if not results:
        return ""

    raw = re.sub(r'[^A-Z]', '', "".join(results).upper())
    print(f"  OCR: '{raw}'")
    return raw

def send_to_web(word):
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

_blank = np.ones((40, 200, 3), dtype=np.uint8) * 255
reader.readtext(_blank, detail=0)
print("Warmup selesai, mulai deteksi...\n")

error_count = 0

while True:
    try:
        word = capture_and_read()
        if word and word != LAST_WORD and len(word) >= 1:
            print(f"✅ Detected: '{word}'")
            send_to_web(word)
            LAST_WORD = word
        error_count = 0

    except Exception as e:
        error_count += 1
        print(f"  Loop error ({error_count}): {e}")
        # Kalau error, paksa recreate sct
        try:
            _sct.close()
        except Exception:
            pass
        _sct = mss.MSS()

        if error_count >= 3:
            print("  ⚠️ Tunggu 3 detik...")
            time.sleep(3)
            error_count = 0

    time.sleep(INTERVAL)