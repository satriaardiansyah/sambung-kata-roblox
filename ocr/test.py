import pyautogui
import requests
import time
import re
import cv2
import numpy as np
import easyocr

reader = easyocr.Reader(['en'], gpu=False)

REGION_HURUF = (1390, 880, 620, 60)  # sesuaikan!
LAST_WORD = ""

def capture_and_read():
    screenshot = pyautogui.screenshot(region=REGION_HURUF)
    screenshot.save("debug_region.png")

    img_cv = cv2.cvtColor(np.array(screenshot), cv2.COLOR_RGB2BGR)
    gray = cv2.cvtColor(img_cv, cv2.COLOR_BGR2GRAY)

    # Step 1: Temukan kotak putih — perlonggar filter aspect ratio
    _, white_mask = cv2.threshold(gray, 200, 255, cv2.THRESH_BINARY)
    contours, _ = cv2.findContours(white_mask, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE)

    boxes = []
    for cnt in contours:
        x, y, w, h = cv2.boundingRect(cnt)
        # ✅ Hapus filter aspect ratio, cukup filter area dan tinggi minimum
        if w * h > 1000 and h > 15:
            boxes.append((x, y, w, h))

    if not boxes:
        print("  ⚠️ Kotak putih tidak ditemukan")
        return ""

    # Ambil kotak TERBESAR (itu pasti kotak kata utama)
    biggest = max(boxes, key=lambda b: b[2] * b[3])
    x0, y0, w0, h0 = biggest

    print(f"  Kotak terbesar: x={x0} y={y0} w={w0} h={h0}")

    # Step 2: Crop dalam kotak
    margin = 6
    roi = img_cv[
        max(0, y0 + margin) : y0 + h0 - margin,
        max(0, x0 + margin) : x0 + w0 - margin
    ]
    cv2.imwrite("debug_roi.png", roi)

    if roi.size == 0:
        print("  ⚠️ ROI kosong")
        return ""

    # Step 3: EasyOCR
    results = reader.readtext(
        roi,
        allowlist='ABCDEFGHIJKLMNOPQRSTUVWXYZ',
        detail=1
    )

    if not results:
        print("  ⚠️ EasyOCR tidak mendeteksi teks")
        return ""

    results.sort(key=lambda r: r[0][0][0])
    raw = "".join(r[1] for r in results).upper()
    raw = re.sub(r'[^A-Z]', '', raw)

    print(f"  Raw OCR: '{raw}'")
    return raw

def send_to_web(word):
    try:
        url = f"http://localhost:8000/auto-input?q={word.lower()}"
        requests.get(url, timeout=1)
        print(f"  Sent: {url}")
    except Exception as e:
        print(f"  Web error: {e}")

print("OCR berjalan... tekan Ctrl+C untuk stop")
print("Cek debug_region.png dan debug_roi.png jika gagal\n")

while True:
    word = capture_and_read()
    if word and word != LAST_WORD and len(word) >= 1:
        print(f"✅ Detected: '{word}'")
        send_to_web(word)
        LAST_WORD = word
    else:
        print(f"  (tidak ada perubahan)")
    time.sleep(0.5)