import pyautogui
import pytesseract
from PIL import ImageFilter, ImageEnhance, ImageOps
import time
import numpy as np
import cv2

REGION = (871, 773, 721, 151)

print("Screenshot dalam 3 detik...")
time.sleep(3)

screenshot = pyautogui.screenshot(region=REGION)
screenshot.save("0_original.png")

# Convert ke OpenCV
img_cv = cv2.cvtColor(np.array(screenshot), cv2.COLOR_RGB2BGR)
gray = cv2.cvtColor(img_cv, cv2.COLOR_BGR2GRAY)

# Threshold Otsu - otomatis pisah foreground/background
_, thresh = cv2.threshold(gray, 0, 255, cv2.THRESH_BINARY_INV + cv2.THRESH_OTSU)
cv2.imwrite("1_thresh.png", thresh)

# Cari bounding box huruf (kontur)
contours, _ = cv2.findContours(thresh, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE)

chars = []
for cnt in contours:
    x, y, w, h = cv2.boundingRect(cnt)
    area = w * h
    # Filter noise kecil, ambil yang proporsional huruf
    if area > 500 and h > 20:
        chars.append((x, y, w, h))

# Urutkan kiri ke kanan
chars.sort(key=lambda c: c[0])
print(f"Karakter terdeteksi: {len(chars)}")

result = ""
for i, (x, y, w, h) in enumerate(chars):
    # Crop + padding
    pad = 5
    roi = thresh[
        max(0, y-pad):y+h+pad,
        max(0, x-pad):x+w+pad
    ]
    
    # Resize besar
    roi_big = cv2.resize(roi, (roi.shape[1]*4, roi.shape[0]*4), 
                          interpolation=cv2.INTER_CUBIC)
    cv2.imwrite(f"2_char_{i}.png", roi_big)
    
    # OCR per karakter
    char = pytesseract.image_to_string(
        roi_big,
        config="--psm 10 -c tessedit_char_whitelist=ABCDEFGHIJKLMNOPQRSTUVWXYZ"
        #        ^^^^ psm 10 = single character
    ).strip().upper()
    
    print(f"  Char {i}: '{char}'")
    result += char

print(f"\nHasil: '{result}'")