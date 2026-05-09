import pytesseract
import pyautogui
import requests
import time
from PIL import Image, ImageFilter, ImageEnhance

# Uncomment sesuai chip Mac kamu:
# pytesseract.pytesseract.tesseract_cmd = "/opt/homebrew/bin/tesseract"  # M1/M2/M3
# pytesseract.pytesseract.tesseract_cmd = "/usr/local/bin/tesseract"     # Intel

REGION = (1000, 850, 500, 60)# ganti dengan koordinat area teks Roblox
LAST_WORD = ""

def capture_and_read():
    screenshot = pyautogui.screenshot(region=REGION)
    img = screenshot.convert("L")
    img = ImageEnhance.Contrast(img).enhance(2.0)
    img = img.filter(ImageFilter.SHARPEN)
    text = pytesseract.image_to_string(img, config="--psm 8 -c tessedit_char_whitelist=abcdefghijklmnopqrstuvwxyz").strip().lower()
    return text

def send_to_web(word):
    try:
        requests.get(f"http://localhost:8000/auto-input?q={word}", timeout=1)
    except:
        pass

print("OCR berjalan... tekan Ctrl+C untuk stop")

while True:
    word = capture_and_read()
    if word and word != LAST_WORD and len(word) >= 2:
        print(f"Detected: {word}")
        send_to_web(word)
        LAST_WORD = word
    time.sleep(0.5)