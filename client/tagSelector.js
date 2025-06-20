export class TagSelector {
  selectedTags;

  #tags;
  #container;
  #tagList;
  #tagSelector;
  #tagSelectorSearch;
  #tagSelectorList;

  #onTagSelectorItemClick(tag) {
    this.addTag(tag);
    this.hide();
  }

  #onTagSelectorSearchInput() {
    const tagResults = this.#tags.filter((item) =>
      item.toLowerCase().includes(this.#tagSelectorSearch.value.toLowerCase()),
    );

    this.#tagSelectorList.textContent = "";
    tagResults.forEach((item) => {
      const tagSelectorItem = document.createElement("button");
      tagSelectorItem.className = "tagSelectorItem";
      tagSelectorItem.textContent = item;
      tagSelectorItem.onclick = () => this.#onTagSelectorItemClick(item);

      this.#tagSelectorList.appendChild(tagSelectorItem);
      this.#tagSelectorList.appendChild(document.createElement("br"));
    });
  }

  #onTagSelectorKeydown(e) {
    const active = document.activeElement;
    const tagSelectorItems =
      this.#tagSelector.querySelectorAll(".tagSelectorItem");

    switch (e.key) {
      case "Escape":
        e.preventDefault();
        this.hide();
        break;

      case "ArrowDown":
        e.preventDefault();
        if (tagSelectorItems.length === 0) return;

        if (!active || !active.classList.contains("tagSelectorItem")) {
          tagSelectorItems[0].focus();
          return;
        }

        const nextItem =
          active.nextElementSibling &&
          active.nextElementSibling.nextElementSibling;
        if (nextItem) nextItem.focus();
        break;

      case "ArrowUp":
        e.preventDefault();
        if (tagSelectorItems.length === 0) return;

        if (!active || !active.classList.contains("tagSelectorItem")) {
          tagSelectorItems[0].focus();
          return;
        }

        const prevItem =
          active.previousElementSibling &&
          active.previousElementSibling.previousElementSibling;
        if (prevItem) prevItem.focus();
        break;

      case "Enter":
        e.preventDefault();
        if (active) active.click();
        break;

      default:
        if (active !== this.#tagSelectorSearch) {
          this.#tagSelectorSearch.focus();
        }
        break;
    }
  }

  constructor(containerId, tags) {
    this.selectedTags = [];

    this.#tags = tags;
    this.#container = document.getElementById(containerId);
    this.#tagList = this.#container.querySelector(".tagList");
    this.#tagSelector = this.#container.querySelector(".tagSelector");
    this.#tagSelectorSearch =
      this.#container.querySelector(".tagSelectorSearch");
    this.#tagSelectorList = this.#container.querySelector(".tagSelectorList");

    this.#tagSelectorSearch.oninput = () => this.#onTagSelectorSearchInput();

    this.#tagSelector.addEventListener("keydown", (e) => {
      this.#onTagSelectorKeydown(e);
    });
  }

  displayBelow(element) {
    const rect = element.getBoundingClientRect();

    this.#tagSelector.style.left = rect.left + "px";
    this.#tagSelector.style.top = rect.bottom + window.scrollY + "px";
    this.#tagSelector.style.display = "block";
    this.#tagSelectorSearch.value = "";

    this.#onTagSelectorSearchInput("");

    this.#tagSelectorSearch.focus();
  }

  hide() {
    this.#tagSelector.style.display = "none";
  }

  containsElement(element) {
    return this.#tagSelector.contains(element);
  }

  addTag(tag) {
    const tagButton = document.createElement("input");
    tagButton.setAttribute("data-tag", tag);
    tagButton.type = "button";
    tagButton.value = "× " + tag;
    tagButton.addEventListener("click", () => {
      this.removeTag(tag);
    });

    if (!this.selectedTags.includes(tag)) {
      this.selectedTags.push(tag);
      this.#tagList.appendChild(tagButton);
    }
  }

  removeTag(tag) {
    for (let item of this.#tagList.children) {
      if (item.getAttribute("data-tag") === tag) {
        item.remove();
        break;
      }
    }

    for (let i = 0; i < this.selectedTags.length; ++i) {
      if (this.selectedTags[i] === tag) {
        this.selectedTags.splice(i, 1);
        break;
      }
    }
  }

  removeAllTags() {
    this.#tagList.textContent = "";
    this.selectedTags = [];
  }
}
