function paginator(pages, currPage) {
    let p = document.getElementById("paginator");
    if (p.childElementCount > 0) {
        return;
    }
    let html = `<li class="page-item"><a class="page-link pager" href="#" data-page="${currPage - 1}">Previous</a></li>`
    for (let i = 0; i < pages; i++) {
        html += `<li class="page-item"><a class="page-link pager" href="#" data-page="${i + 1}">${i + 1}</a></li>`
    }
    html += `<li class="page-item"><a class="page-link pager" href="#" data-page="${currPage + 1}">Next</a></li>`
    p.innerHTML = html;
    p.childNodes[currentPage].firstChild.classList.add("active");

    let pgBtns = document.getElementsByClassName("pager");
    for (let j = 0; j < pgBtns.length; j++) {
        pgBtns[j].addEventListener("click", function (evt) {
            let pageNo = evt.target.getAttribute("data-page");
            if (pageNo > 0 && pageNo <= pages) {
                currentPage = pageNo;
                let p = document.getElementById("paginator");
                // remove active attribute from all pages
                p.childNodes.forEach(el => el.firstChild.classList.remove("active"));
                // get data from the server
                updateTable(pageSize, currentPage);
                // set active attribute to the page that is currently active
                p.childNodes[parseInt(currentPage, 10)].firstChild.classList.add("active");
                // update previous/next page attributes
                p.childNodes[0].firstChild.setAttribute("data-page", parseInt(currentPage, 10) - 1);
                p.childNodes[pages + 1].firstChild.setAttribute("data-page", parseInt(currentPage, 10) + 1);
            }
        });
    }
}
