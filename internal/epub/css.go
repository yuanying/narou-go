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
}`
