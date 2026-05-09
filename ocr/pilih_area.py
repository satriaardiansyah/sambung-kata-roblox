import tkinter as tk
import pyautogui
from PIL import ImageTk, Image

# Screenshot seluruh layar dulu
screenshot = pyautogui.screenshot()
screen_w, screen_h = screenshot.size

start_x = start_y = end_x = end_y = 0
rect_id = None

def on_press(e):
    global start_x, start_y, rect_id
    start_x, start_y = e.x, e.y
    rect_id = canvas.create_rectangle(start_x, start_y, start_x, start_y,
                                       outline="red", width=2)

def on_drag(e):
    canvas.coords(rect_id, start_x, start_y, e.x, e.y)

def on_release(e):
    global end_x, end_y
    end_x, end_y = e.x, e.y
    w = end_x - start_x
    h = end_y - start_y
    print(f"\n✅ Koordinat area:")
    print(f"REGION = ({start_x}, {start_y}, {w}, {h})")
    
    # Preview area yang dipilih
    cropped = screenshot.crop((start_x, start_y, end_x, end_y))
    cropped.save("preview_region.png")
    print(f"Preview disimpan ke preview_region.png")
    root.destroy()

root = tk.Tk()
root.attributes("-fullscreen", True)
root.attributes("-alpha", 0.4)  # transparan biar keliatan layar
root.configure(bg="black")

# Tampilkan screenshot sebagai background
img = ImageTk.PhotoImage(screenshot.resize((screen_w, screen_h)))
canvas = tk.Canvas(root, width=screen_w, height=screen_h, cursor="cross")
canvas.pack()
canvas.create_image(0, 0, anchor="nw", image=img)

label = tk.Label(root, text="Klik dan drag untuk pilih area teks Roblox — tekan ESC untuk batal",
                 fg="white", bg="black", font=("Arial", 14))
label.place(x=20, y=20)

canvas.bind("<ButtonPress-1>", on_press)
canvas.bind("<B1-Motion>", on_drag)
canvas.bind("<ButtonRelease-1>", on_release)
root.bind("<Escape>", lambda e: root.destroy())

root.mainloop()