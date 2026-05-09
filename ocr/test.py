import pytesseract
import pyautogui
import requests
import time
import re
import cv2
import numpy as np
from PIL import Image, ImageEnhance, ImageOps

pytesseract.pytesseract.tesseract_cmd = "/opt/homebrew/bin/tesseract"  # M1/M2/M3
# pytesseract.pytesseract.tesseract_cmd = "/usr/local/bin/tesseract"   # Intel

# Sesuaikan region ke area "Hurufnya adalah: SF" (teks putih di bawah karakter)
# Dari screenshot kamu, area ini sekitar y=720 di layar Roblox
REGION_HURUF = (1420, 880, 500, 60)  # (x, y, width, height) — sesuaikan!

LAST_WORD = ""

def capture_and_read():
    screenshot = pyautogui.screenshot(region=REGION_HURUF)
    screenshot.save("debug_region.png")  # simpan untuk debug

    # Convert ke numpy untuk OpenCV
    img_cv = cv2.cvtColor(np.array(screenshot), cv2.COLOR_RGB2BGR)
    gray = cv2.cvtColor(img_cv, cv2.COLOR_BGR2GRAY)

    # Adaptive threshold — tahan terhadap background warna apapun
    thresh = cv2.adaptiveThreshold(
        gray, 255,
        cv2.ADAPTIVE_THRESH_GAUSSIAN_C,
        cv2.THRESH_BINARY,
        blockSize=31,  # ukuran area lokal, harus ganjil
        C=10
    )
    cv2.imwrite("debug_thresh.png", thresh)

    # Resize 3x biar OCR lebih akurat
    h, w = thresh.shape
    big = cv2.resize(thresh, (w*3, h*3), interpolation=cv2.INTER_CUBIC)

    # OCR seluruh teks dulu
    raw = pytesseract.image_to_string(
        big,
        config="--psm 7 -c tessedit_char_whitelist=ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz :"
    ).strip()

    print(f"  Raw OCR: '{raw}'")

    # Ekstrak huruf setelah "adalah:" pakai regex
    match = re.search(r'adalah[:\s]+([A-Z]{1,3})', raw, re.IGNORECASE)
    if match:
        return match.group(1).upper()

    # Fallback: ambil semua huruf kapital satu per satu
    caps = re.findall(r'[A-Z]', raw)
    if caps:
        # Hapus duplikat berurutan & karakter noise umum (I, T, L sering muncul palsu)
        NOISE_CHARS = {'I', 'T', 'L', '1', '|'} if len(caps) > 2 else set()
        caps = [c for c in caps if c not in NOISE_CHARS]
        # Ambil maks 3 karakter pertama
        return "".join(caps[:3])

    return ""

def send_to_web(word):
    try:
        url = f"http://localhost:8000/auto-input?q={word.lower()}"
        requests.get(url, timeout=1)
        print(f"  Sent to web: {url}")
    except Exception as e:
        print(f"  Web error: {e}")

print("OCR berjalan... tekan Ctrl+C untuk stop")
print("Cek debug_region.png dan debug_thresh.png jika gagal\n")

while True:
    word = capture_and_read()
    if word and word != LAST_WORD and len(word) >= 1:
        print(f"✅ Detected: '{word}'")
        send_to_web(word)
        LAST_WORD = word
    else:
        print(f"  (tidak ada perubahan atau kosong)")
    time.sleep(0.5)