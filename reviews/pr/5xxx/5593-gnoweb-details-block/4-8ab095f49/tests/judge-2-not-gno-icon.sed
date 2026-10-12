# judge-2: applied to a copy of gno.land/pkg/gnoweb/public/main.css at 8ab095f49,
# then solo-finder-icon-rotated.html rerun: every gno-icon reports transform none.
s/\.gno-details>summary>svg/.gno-details>summary>svg:not(.gno-icon)/g
s/\.gno-details\[open\]>summary>svg/.gno-details[open]>summary>svg:not(.gno-icon)/g
