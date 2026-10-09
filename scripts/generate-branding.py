#!/usr/bin/env python3
"""Generate the shared, font-independent television mark and geometric wordmark."""
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
BLUE = '#246BDE'
WHITE = '#DFE3EA'
# Custom squared/rounded lettering follows the approved Power Play direction.
GLYPHS = {
'A': (86, 'M5 70Q0 70 3 63L28 7Q31 0 39 0H47Q55 0 58 7L83 63Q86 70 81 70H66L59 53H27L20 70Z M34 37H52L43 15Z'),
'l': (22, 'M3 0H19V70H3Z'),
'r': (52, 'M2 70V22H18V28Q24 20 35 20H50V36H34Q18 36 18 49V70Z'),
'e': (65, 'M23 20H43Q63 20 63 40V51H18V52Q18 57 25 57H60V70H23Q2 70 2 49V41Q2 20 23 20Z M18 39H47Q47 32 40 32H25Q18 32 18 39Z'),
'a': (66, 'M7 20H43Q64 20 64 41V70H24Q2 70 2 53Q2 37 24 37H48Q48 33 40 33H7Z M25 49Q18 49 18 53Q18 58 25 58H48V49Z'),
'd': (68, 'M50 0H66V70H24Q2 70 2 49V41Q2 20 24 20H50Z M25 35Q18 35 18 42V48Q18 55 25 55H50V35Z'),
'y': (66, 'M2 20H18V48Q18 55 25 55H48V20H64V68Q64 88 42 88H12V73H41Q48 73 48 68V67H24Q2 67 2 48Z'),
'O': (81, 'M24 0H57Q79 0 79 22V48Q79 70 57 70H24Q2 70 2 48V22Q2 0 24 0Z M25 17Q19 17 19 24V46Q19 53 25 53H56Q62 53 62 46V24Q62 17 56 17Z'),
'n': (68, 'M2 70V20H18V25Q26 20 37 20H43Q66 20 66 43V70H50V44Q50 35 40 35H29Q18 35 18 44V70Z'),
}
MARK = [
 ('none', BLUE, 9, 'M29 24L45 40L61 24 M70 40H30Q15 40 15 55V80Q15 95 30 95H74Q89 95 89 80V56'),
 (BLUE, None, 0, 'M98 42A11 11 0 1 1 76 42A11 11 0 1 1 98 42Z'),
 (WHITE, None, 0, 'M42 55Q39 53 39 58V79Q39 84 43 82L61 72Q66 69 61 66Z'),
]

def mark_svg():
 return ''.join(f'<path d="{d}" fill="{fill}"'+(f' stroke="{stroke}" stroke-width="{width}" stroke-linecap="round" stroke-linejoin="round"' if stroke else '')+'/>' for fill,stroke,width,d in MARK)

def letters_svg():
 x=0;out=[]
 for ch in 'AlreadyOn':
  width,d=GLYPHS[ch];color=BLUE if ch in 'On' else WHITE
  out.append(f'<path transform="translate({x} 0)" d="{d}" fill="{color}" fill-rule="evenodd"/>');x+=width+3
 return ''.join(out),x

def svg(body,viewbox,title):
 return f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="{viewbox}" role="img" aria-label="{title}"><title>{title}</title>{body}</svg>\n'

letters,width=letters_svg()
brand=ROOT/'web/public/branding';brand.mkdir(parents=True,exist_ok=True)
icon=svg(mark_svg(),'0 0 108 108','AlreadyOn television icon')
wordmark=svg(letters,f'0 0 {width} 90','AlreadyOn')
# Make the icon 5% taller than the wordmark, preserving aspect ratio.
# The tagline sits below that shared line and does not affect icon sizing.
def lockup(tagline=False):
 top = 5 if tagline else 14
 wordmark_height = 88*.95
 height = wordmark_height*1.05
 icon_top = top+wordmark_height-height
 scale = height/80
 left = 87.5*scale+18
 mark = f'<g transform="translate(0 {icon_top}) scale({scale}) translate(-10.5 -19.5)">{mark_svg()}</g>'
 text = f'<g transform="translate({left} {top}) scale(.95)">{letters}</g>'
 if tagline:
  text += f'<text x="{left}" y="110.8" fill="#8B93A3" font-family="system-ui,sans-serif" font-size="25">Your library, tuned to you</text>'
 return mark+text,left+width*.95

compact_body,compact_width=lockup()
full_body,full_width=lockup(True)
compact=svg(compact_body,f'0 0 {compact_width:g} 108','AlreadyOn')
full=svg(full_body,f'0 0 {full_width:g} 132','AlreadyOn — Your library, tuned to you')
doc_header = compact.replace('<title>', '<rect width="100%" height="100%" rx="10" fill="#12141A"/><title>', 1).replace('<svg ', '<svg width="240" height="38" ', 1)
banner = svg('<rect width="100%" height="100%" rx="10" fill="#12141A"/><g transform="translate(20 20)">'+full_body+'</g>',f'0 0 {full_width+40:g} 172','AlreadyOn — Your library, tuned to you').replace('<svg ', '<svg width="720" height="168" ', 1)
for name,content in [('icon.svg',icon),('wordmark.svg',wordmark),('header.svg',compact),('logo.svg',full),('banner.svg',banner),('header-doc.svg',doc_header)]: (brand/name).write_text(content)
(ROOT/'web/public/favicon.svg').write_text(icon)
# Native vectors use the same paths, keeping launcher and in-app art consistent.
res=ROOT/'apps/android/app/src/main/res'
def vector(body,w,h):
 return f'<?xml version="1.0" encoding="utf-8"?>\n<!-- Generated from scripts/generate-branding.py; edit the shared paths there. -->\n<vector xmlns:android="http://schemas.android.com/apk/res/android" android:width="{w}dp" android:height="{h}dp" android:viewportWidth="{w}" android:viewportHeight="{h}">{body}</vector>\n'
paths=''.join(f'<path android:pathData="{d}" android:fillColor="{fill if fill!="none" else "#00000000"}"'+(f' android:strokeColor="{stroke}" android:strokeWidth="{sw}" android:strokeLineCap="round" android:strokeLineJoin="round"' if stroke else '')+'/>' for fill,stroke,sw,d in MARK)
(res/'drawable/ic_brand.xml').write_text(vector(paths,108,108))
(res/'drawable/ic_launcher_foreground.xml').write_text(vector(f'<group android:scaleX="0.65" android:scaleY="0.65" android:translateX="18.9" android:translateY="18.9">{paths}</group>',108,108))
x=0;native=[]
for ch in 'AlreadyOn':
 w,d=GLYPHS[ch];native.append(f'<group android:translateX="{x}"><path android:pathData="{d}" android:fillColor="{BLUE if ch in "On" else WHITE}" android:fillType="evenOdd"/></group>');x+=w+3
(res/'drawable/brand_wordmark.xml').write_text(vector(''.join(native),width,90))
# Older launchers need an ordinary drawable; adaptive launchers use their own mask.
(res/'mipmap-anydpi').mkdir(exist_ok=True)
(res/'mipmap-anydpi/ic_launcher.xml').write_text(vector('<path android:fillColor="#12141A" android:pathData="M0 0H108V108H0Z"/>'+f'<group android:scaleX="0.8" android:scaleY="0.8" android:translateX="10.8" android:translateY="10.8">{paths}</group>',108,108))
print('Shared SVG and Android branding generated.')

for label, name in [('Install', 'install'), ('Develop', 'develop'), ('Contribute', 'contribute'), ('GPL-3.0', 'license')]:
 badge = svg('<rect width="112" height="30" rx="6" fill="#246BDE"/><text x="56" y="20" text-anchor="middle" fill="#FFFFFF" font-family="system-ui,sans-serif" font-size="13">'+label+'</text>', '0 0 112 30', label).replace('<svg ', '<svg width="112" height="30" ', 1)
 (brand/(name+'-badge.svg')).write_text(badge)
