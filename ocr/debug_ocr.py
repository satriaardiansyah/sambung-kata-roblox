import tkinter as tk
import pytesseract
import pyautogui
from PIL import ImageFilter, ImageEnhance
import threading
import time

# ✅ Ganti dengan koordinat kamu
REGION = (990, 740, 527, 129)

detected_text = ""

def ocr_loop(label):
    global detected_text
    while True:
        try:
            screenshot = pyautogui.screenshot(region=REGION)
            img = screenshot.convert("L")
            img = ImageEnhance.Contrast(img).enhance(2.0)
            img = img.filter(ImageFilter.SHARPEN)
            text = pytesseract.image_to_string(
                img,
                config="--psm 8 -c tessedit_char_whitelist=abcdefghijklmnopqrstuvwxyz"
            ).strip().lower()

            detected_text = text if text else "(tidak terdeteksi)"
            label.config(text=f"Terdeteksi: {detected_text}")
        except Exception as e:
            label.config(text=f"Error: {e}")
        time.sleep(0.5)

def main():
    root = tk.Tk()
    root.title("OCR Debug")
    root.attributes("-topmost", True)
    root.attributes("-alpha", 0.9)
    root.configure(bg="black")
    root.resizable(False, False)

    # Posisikan window debug tepat di bawah REGION
    win_x = REGION[0]
    win_y = REGION[1] + REGION[3] + 5
    root.geometry(f"400x60+{win_x}+{win_y}")

    label = tk.Label(
        root,
        text="Mendeteksi...",
        fg="#00ff88",
        bg="black",
        font=("Courier", 13, "bold"),
        pady=8
    )
    label.pack(fill="x")

    # Overlay kotak merah — pakai -transparent (Mac)
    overlay = tk.Toplevel(root)
    overlay.attributes("-topmost", True)
    overlay.attributes("-transparent", True)   # ← fix untuk Mac
    overlay.overrideredirect(True)
    overlay.geometry(f"{REGION[2]}x{REGION[3]}+{REGION[0]}+{REGION[1]}")

    canvas = tk.Canvas(
        overlay,
        width=REGION[2],
        height=REGION[3],
        bg="systemTransparent",               # ← transparan di Mac
        highlightthickness=0
    )
    canvas.pack()

    # Kotak merah border saja, tanpa fill
    canvas.create_rectangle(
        1, 1, REGION[2]-1, REGION[3]-1,
        outline="red", width=2, fill=""
    )

    # Jalankan OCR di thread terpisah
    t = threading.Thread(target=ocr_loop, args=(label,), daemon=True)
    t.start()

    root.mainloop()

if __name__ == "__main__":
    main()