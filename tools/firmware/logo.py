"""Render the existing Maco SVG as the uncompressed 24-bit EDK II bitmap."""
import io
import sys

import cairosvg
from PIL import Image

png = cairosvg.svg2png(url=sys.argv[1], output_width=256, output_height=256)
logo = Image.open(io.BytesIO(png)).convert("RGBA")
background = Image.new("RGB", logo.size, "black")
background.paste(logo, mask=logo.getchannel("A"))
background.save(sys.argv[2], "BMP")
