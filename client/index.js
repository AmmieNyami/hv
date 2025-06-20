import { TagSelector } from "./tagSelector.js";
import { TagSetManager } from "./tagSetManager.js";
import * as api from "./api.js";

const config = await fetch("/config.json").then((res) => res.json());

function setupAddTagButton(button, tagSelector) {
  button.onclick = (e) => {
    tagSelector.displayBelow(e.target);
  };

  document.addEventListener("click", (e) => {
    if (!tagSelector.containsElement(e.target) && e.target !== button) {
      tagSelector.hide();
    }
  });
}

async function onLogoutButtonClick() {
  await api.logout();
  window.location.replace("/login");
}

function makeBlackImage(width, height) {
  const canvas = document.createElement("canvas");
  canvas.width = width;
  canvas.height = height;

  const ctx = canvas.getContext("2d");
  ctx.fillStyle = "rgba(0, 0, 0, 255)";
  ctx.fillRect(0, 0, width, height);

  return canvas.toDataURL();
}

async function displaySearchResults(query, tags, antiTags, page) {
  const searchResults = document.getElementById("searchResults");
  searchResults.textContent = "Loading...";

  const pageSelector = document.getElementById("pageSelector");
  pageSelector.textContent = "";

  let results = await api.search(query, tags, antiTags, 21, page);
  if (!results.entries) {
    searchResults.textContent = "No results.";
    return;
  }

  searchResults.textContent = "";
  for (let entry of results.entries) {
    // Main metadata
    let searchResult;
    let searchResultImage;
    {
      searchResult = document.createElement("div");
      searchResult.className = "searchResult";

      searchResultImage = document.createElement("img");
      searchResultImage.className = "searchResultImage";
      searchResultImage.src = makeBlackImage(120, 167);

      const searchResultTitle = document.createElement("div");
      searchResultTitle.textContent = entry.title;
      searchResultTitle.className = "searchResultTitle";

      const searchResultSubtitle = document.createElement("div");
      searchResultSubtitle.textContent = entry.subtitle;
      searchResultSubtitle.className = "searchResultSubtitle";

      searchResult.appendChild(searchResultImage);
      searchResult.appendChild(searchResultTitle);
      searchResult.appendChild(searchResultSubtitle);
    }

    // Expanded metadata
    let expandedMeta;
    let expandedMetaImage;
    {
      expandedMeta = document.createElement("div");
      expandedMeta.className = "searchResultExpandedMeta";
      expandedMeta.style.display = "none";

      expandedMetaImage = document.createElement("img");
      expandedMetaImage.src = makeBlackImage(1, 1);
      expandedMetaImage.className = "searchResultExpandedMetaImage";

      const expandedMetaTitle = document.createElement("div");
      expandedMetaTitle.textContent = entry.title;
      expandedMetaTitle.className = "searchResultExpandedMetaTitle";

      const expandedMetaSubtitle = document.createElement("div");
      expandedMetaSubtitle.textContent = "Subtitle: " + entry.subtitle;
      expandedMetaSubtitle.className = "searchResultExpandedMetaSubtitle";

      const uploadedAt = document.createElement("div");
      uploadedAt.textContent =
        "Uploaded at " + new Date(entry.upload_date).toDateString();

      const externalRating = document.createElement("div");
      externalRating.className = "searchResultExpandedMetaExternalRating";
      externalRating.textContent = "Rated " + entry.external_rating;

      const languages = document.createElement("div");
      languages.className = "searchResultExpandedMetaLanguages";
      {
        if (entry.languages.length !== 0) {
          languages.textContent = "Languages: ";
          for (let [index, language] of entry.languages.entries()) {
            languages.textContent += language;
            if (index !== entry.languages.length - 1)
              languages.textContent += ", ";
          }
          languages.textContent = languages.textContent.trim();
        }
      }

      const readButton = document.createElement("button");
      readButton.textContent = "Read";
      readButton.className = "searchResultExpandedMetaReadButton";
      readButton.onclick = () => {
        window.open(`/viewer?doujinId=${entry.id}`, "_blank");
      };

      // Append the elements in the main metadata
      expandedMeta.appendChild(expandedMetaImage);
      expandedMeta.appendChild(expandedMetaTitle);
      expandedMeta.appendChild(expandedMetaSubtitle);

      // Append elements exclusive to the expanded metadata
      expandedMeta.appendChild(uploadedAt);
      expandedMeta.appendChild(externalRating);
      expandedMeta.appendChild(languages);
      expandedMeta.appendChild(readButton);
    }

    const capePageId = entry.pages[0][1];
    if (capePageId !== 0) {
      await api.getPage(capePageId).then((pageImageURL) => {
        searchResultImage.src = pageImageURL;
        expandedMetaImage.src = pageImageURL;
      });
    }

    searchResult.onclick = () => {
      expandedMeta.style.display =
        expandedMeta.style.display === "none" ? "block" : "none";
    };

    searchResults.appendChild(searchResult);
    searchResults.appendChild(expandedMeta);
  }

  const goToPage = (page) => {
    if (page < 1) return;
    if (page > results.total_pages) return;
    displaySearchResults(query, tags, antiTags, page);
  };

  if (page - 3 > 1) {
    const beginButton = document.createElement("button");
    beginButton.textContent = "<<";
    beginButton.onclick = () => goToPage(1);
    pageSelector.appendChild(beginButton);
  }

  if (page > 1) {
    const previousButton = document.createElement("button");
    previousButton.textContent = "<";
    previousButton.onclick = () => goToPage(page - 1);
    pageSelector.appendChild(previousButton);
  }

  for (let i = Math.max(1, page - 3); i < page; ++i) {
    const pageButton = document.createElement("button");
    pageButton.textContent = `${i}`;
    pageButton.onclick = () => goToPage(i);
    pageSelector.appendChild(pageButton);
  }

  for (let i = page + 1; i <= Math.min(page + 3, results.total_pages); ++i) {
    const pageButton = document.createElement("button");
    pageButton.textContent = `${i}`;
    pageButton.onclick = () => goToPage(i);
    pageSelector.appendChild(pageButton);
  }

  if (page < results.total_pages) {
    const nextButton = document.createElement("button");
    nextButton.textContent = ">";
    nextButton.onclick = () => goToPage(page + 1);
    pageSelector.appendChild(nextButton);
  }

  if (page + 3 < results.total_pages) {
    const endButton = document.createElement("button");
    endButton.textContent = ">>";
    endButton.onclick = () => goToPage(results.total_pages);
    pageSelector.appendChild(endButton);
  }
}

await (async () => {
  if (window.location.host !== new URL(config.api_url).host) {
    document.getElementById("body").textContent =
      `The frontend should only be accessed through the API URL (${config.api_url}).`;
    return;
  }

  const needsLogin = await api.needsLogin();
  if (needsLogin) {
    window.location.replace("/login");
    return;
  }

  const username = await api.getUsername();

  const usernameDisplay = document.getElementById("usernameDisplay");
  usernameDisplay.textContent = "Logged in as " + username;

  const logoutButton = document.getElementById("logoutButton");
  logoutButton.onclick = onLogoutButtonClick;

  const tags = await api.getTags();

  const tagSelector = new TagSelector("tagSelector", tags);
  const addTagButton = document.getElementById("addTagButton");
  setupAddTagButton(addTagButton, tagSelector);

  const antiTagSelector = new TagSelector("antiTagSelector", tags);
  const addAntiTagButton = document.getElementById("addAntiTagButton");
  setupAddTagButton(addAntiTagButton, antiTagSelector);

  const tagSets = await api.getTagSets();
  const tagSetManager = new TagSetManager(
    tagSets,
    tagSelector,
    antiTagSelector,
  );

  const addNewTagSet = document.getElementById("addNewTagSet");
  addNewTagSet.onclick = async () => {
    tagSetManager.createTagSet();
  };

  const removeTagSet = document.getElementById("removeTagSet");
  removeTagSet.onclick = async () => {
    tagSetManager.removeTagSet();
  };

  const updateTagSet = document.getElementById("updateTagSet");
  updateTagSet.onclick = async () => {
    tagSetManager.changeTagSet();
  };

  const searchField = document.getElementById("searchField");
  searchField.addEventListener("keydown", async (e) => {
    if (e.key === "Enter") {
      e.preventDefault();
      await displaySearchResults(
        searchField.value,
        tagSelector.selectedTags,
        antiTagSelector.selectedTags,
        1,
      );
    }
  });
})();
