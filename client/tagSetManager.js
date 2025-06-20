import { error } from "./error.js";
import * as api from "./api.js";

export class TagSetManager {
  #tagSelector;
  #antiTagSelector;

  #tagSets;

  #tagSetSelector;

  constructor(tagSets, tagSelector, antiTagSelector) {
    this.#tagSelector = tagSelector;
    this.#antiTagSelector = antiTagSelector;

    this.#tagSets = tagSets;

    this.#tagSetSelector = document.getElementById("tagSetSelector");
    this.#tagSetSelector.addEventListener("change", () => {
      const tagSetIndex = this.#tagSetSelector.selectedIndex - 1;
      if (tagSetIndex === -1) return;

      const tagSet = this.#tagSets[tagSetIndex];

      this.#tagSelector.removeAllTags();
      this.#antiTagSelector.removeAllTags();

      for (const tag of tagSet.tags) {
        this.#tagSelector.addTag(tag);
      }

      for (const tag of tagSet.anti_tags) {
        this.#antiTagSelector.addTag(tag);
      }
    });

    this.#render();
    this.#clearSelection();
  }

  #render() {
    const tagSetToSelectorOptionText = (tagSet) => {
      let textContent = "";
      for (const tag of tagSet.tags) {
        textContent += "+" + tag + " ";
      }

      for (const tag of tagSet.anti_tags) {
        textContent += "-" + tag + " ";
      }
      textContent = textContent.trim();

      let maxLength = Math.floor((window.innerWidth / 630) * 30);
      if (maxLength > 50) maxLength = 50;
      if (maxLength - 3 > 0) {
        textContent = textContent.slice(0, maxLength - 3).trim() + "...";
      }

      return textContent;
    };

    this.#tagSetSelector.textContent = "";

    const option = document.createElement("option");
    option.disabled = true;
    option.textContent =
      this.#tagSets.length === 0 ? "No tag sets" : "Select tag set";
    this.#tagSetSelector.appendChild(option);

    for (const tagSet of this.#tagSets) {
      const option = document.createElement("option");
      option.textContent = tagSetToSelectorOptionText(tagSet);
      this.#tagSetSelector.appendChild(option);
    }
  }

  #select(index) {
    this.#tagSetSelector.selectedIndex = index + 1;
  }

  #clearSelection() {
    this.#select(-1);
  }

  async createTagSet() {
    if (
      this.#tagSelector.selectedTags.length === 0 &&
      this.#antiTagSelector.selectedTags.length === 0
    ) {
      error("Please select some tags before creating a new tag set");
    }

    const tagSetId = await api.createTagSet(
      this.#tagSelector.selectedTags,
      this.#antiTagSelector.selectedTags,
    );
    const tagSet = {
      id: tagSetId,
      tags: [...this.#tagSelector.selectedTags],
      anti_tags: [...this.#antiTagSelector.selectedTags],
    };
    const tagSetIndex = this.#tagSets.length;

    this.#tagSets.push(tagSet);

    this.#render();
    this.#select(tagSetIndex);

    return tagSetIndex;
  }

  async changeTagSet() {
    const tagSetIndex = this.#tagSetSelector.selectedIndex - 1;
    if (tagSetIndex === -1) error("Please select a tag set to update");

    if (
      this.#tagSelector.selectedTags.length === 0 &&
      this.#antiTagSelector.selectedTags.length === 0
    ) {
      error("Please select some tags before updating the current tag set");
    }

    const tagSetId = this.#tagSets[tagSetIndex].id;
    await api.changeTagSet(
      tagSetId,
      this.#tagSelector.selectedTags,
      this.#antiTagSelector.selectedTags,
    );

    this.#tagSets[tagSetIndex] = {
      id: tagSetId,
      tags: [...this.#tagSelector.selectedTags],
      anti_tags: [...this.#antiTagSelector.selectedTags],
    };

    this.#render();
    this.#select(tagSetIndex);

    return tagSetIndex;
  }

  async removeTagSet() {
    const tagSetIndex = this.#tagSetSelector.selectedIndex - 1;
    if (tagSetIndex === -1) error("Please select a tag set to remove");

    await api.deleteTagSet(this.#tagSets[tagSetIndex].id);

    this.#tagSets.splice(tagSetIndex, 1);

    this.#render();
    this.#clearSelection();
  }
}
