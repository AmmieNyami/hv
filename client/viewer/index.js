import * as api from "../api.js";
import { error } from "../error.js";

await (async () => {
  const searchParams = new URLSearchParams(window.location.search);
  let doujinId = searchParams.get("doujinId");
  if (!doujinId) {
    error("Please specify a doujin ID in the `doujinId` URL parameter.");
    return;
  }
  doujinId = Number(doujinId);

  const doujin = await api.getDoujin(doujinId);

  const pages = doujin.pages.sort((a, b) => a[0] - b[0]);
  let currentPage = 0;

  const pageImage = document.getElementById("pageImage");

  const renderCurrentPage = async () => {
    pageImage.src = await api.getPage(pages[currentPage][1]);
  };

  pageImage.addEventListener("click", (e) => {
    const imgRect = pageImage.getBoundingClientRect();
    const imgCenterX = imgRect.left + imgRect.width / 2;

    const clickX = e.clientX;

    if (clickX < imgCenterX) {
      if (currentPage > 0) {
        currentPage -= 1;
        renderCurrentPage();
      }
    } else {
      if (currentPage < pages.length - 1) {
        currentPage += 1;
        renderCurrentPage();
      }
    }
  });

  pageImage.addEventListener("mousemove", (e) => {
    const pageImageRect = pageImage.getBoundingClientRect();
    const centerX = pageImageRect.left + pageImageRect.width / 2;

    if (e.clientX < centerX) {
      pageImage.style.cursor = "url(/assets/cursors/cursor-left.png), auto";
    } else {
      pageImage.style.cursor = "url(/assets/cursors/cursor-right.png), auto";
    }
  });

  pageImage.addEventListener("mouseleave", () => {
    pageImage.style.cursor = "default";
  });

  renderCurrentPage();
})();
