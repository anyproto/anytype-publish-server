function initToggles() {
    const toggles = document.querySelectorAll(".textToggle");
    toggles.forEach(t => {
        t.addEventListener("click", function() {
            t.classList.toggle("isToggled");
        })
    })

}

function initLatex() {
    const katex = window.katex
    const latexBlocks = document.querySelectorAll(".isLatex .content");
    const trustFn = context => {
        return [ '\\url', '\\href', '\\includegraphics' ].includes(context.command)
    }
    latexBlocks.forEach(b => {
        const latexFormula = b.innerText
        let html = ""
        try {
            html = katex.renderToString(latexFormula, {
                displayMode: true,
                strict: false,
                throwOnError: true,
                output: 'html',
                fleqn: true,
                trust: trustFn,
            });
        } catch (e) {
            console.error(e);
            if (e instanceof katex.ParseError) {
                html = `<div class="error">Error parsing LaTeX</div>`;
            }
        };

        b.innerHTML = html

    })

}

function initMermaid() {
    mermaid.initialize({ startOnLoad: true });
}

function initGraphviz() {
    const gphBlocks = document.querySelectorAll(".isGraphviz");
    gphBlocks.forEach(b => {
        const gphFormula = window.svgSrc[b.id].content
        try {
            const viz = new Viz()
            viz.renderSVGElement(gphFormula).then(svg => {
                parent = b.querySelector(".content")
                parent.appendChild(svg);
            }, err => {
                console.error("viz error:",err)
            });
        } catch (e) {
            console.error("viz error:",e);
        };
    })
}

window.svgSrc = {}

document.addEventListener("DOMContentLoaded", function() {
    const initFns = [initToggles, initLatex, initMermaid, initGraphviz]
    initFns.forEach(f => {
        setTimeout(_ => {
            try {
                f()
            } catch (e) {
                console.error(`error executing init function "${f.name}":`, e)
            }
        })
    })
});
