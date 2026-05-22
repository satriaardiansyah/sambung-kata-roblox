# pilih_area.py
import tkinter as tk
import mss
import mss.tools
from PIL import ImageTk, Image

print("Mengambil screenshot...")

with mss.MSS() as sct:
    monitor = sct.monitors[1]
    shot = sct.grab(monitor)
    mss.tools.to_png(shot.rgb, shot.size, output="fullscreen_temp.png")
    screen_w = monitor["width"]
    screen_h = monitor["height"]

start_x = start_y = end_x = end_y = 0
rect_id = None

def on_press(e):
    global start_x, start_y, rect_id
    start_x, start_y = e.x, e.y
    rect_id = canvas.create_rectangle(start_x, start_y, start_x, start_y,
                                      outline="red", width=3)

def on_drag(e):
    canvas.coords(rect_id, start_x, start_y, e.x, e.y)

def on_release(e):
    w = abs(e.x - start_x)
    h = abs(e.y - start_y)
    x = min(start_x, e.x)
    y = min(start_y, e.y)
    print(f"\n✅ Koordinat area:")
    print(f"REGION_HURUF = ({x}, {y}, {w}, {h})")
    root.destroy()

root = tk.Tk()
root.attributes("-fullscreen", True)
root.attributes("-topmost", True)

# Tampilkan screenshot tanpa overlay gelap
img = Image.open("fullscreen_temp.png").resize((screen_w, screen_h))
tk_img = ImageTk.PhotoImage(img)

canvas = tk.Canvas(root, width=screen_w, height=screen_h,
                   cursor="cross", highlightthickness=0)
canvas.pack()
canvas.create_image(0, 0, anchor="nw", image=tk_img)

# Label petunjuk tipis di atas saja
label = tk.Label(root,
    text="🖱 Klik dan drag di area kotak huruf — ESC batal",
    fg="white", bg="red", font=("Arial", 13, "bold"), pady=6)
label.place(x=0, y=0, width=screen_w)

canvas.bind("<ButtonPress-1>", on_press)
canvas.bind("<B1-Motion>", on_drag)
canvas.bind("<ButtonRelease-1>", on_release)
root.bind("<Escape>", lambda e: root.destroy())

root.mainloop()