package epub

// VerticalCSS is the default stylesheet for vertical Japanese text.
const VerticalCSS = `body {
  writing-mode: vertical-rl;
  -epub-writing-mode: vertical-rl;
  line-height: 1.6;
}

span.tcy {
  text-combine-upright: all;
  -webkit-text-combine: horizontal;
}

em.emphasisDots {
  text-emphasis: sesame;
  -webkit-text-emphasis: sesame;
  font-style: normal;
}

span.strikethrough {
  text-decoration: line-through;
}

p.half-indent {
  text-indent: -0.5em;
}

div.cover {
  width: 100%;
  height: 100%;
  text-align: center;
}

div.cover img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

p.cover-title {
  font-size: 1.5em;
  font-weight: bold;
  margin-top: 1em;
}

p.cover-author {
  font-size: 1em;
  margin-top: 0.5em;
}

div.chapter-page {
  display: flex;
  justify-content: center;
  align-items: center;
  width: 100%;
  height: 100%;
  text-align: center;
}`
