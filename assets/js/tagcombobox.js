(function () {
  "use strict";

  // --- Chip creation ---
  function createChip(name, disabled) {
    var chip = document.createElement("div");
    chip.setAttribute("data-tagcombobox-chip", name);
    chip.className =
      "inline-flex items-center gap-2 rounded-md border px-2.5 py-0.5 text-xs font-semibold transition-colors border-transparent bg-primary text-primary-foreground";
    chip.innerHTML =
      '<span>' + escapeHTML(name) + '</span>' +
      '<button type="button" class="ml-1 text-current hover:text-destructive disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer" data-tagcombobox-remove="' + escapeAttr(name) + '"' + (disabled ? " disabled" : "") + '>' +
      '<svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3 pointer-events-none" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"/></svg>' +
      '</button>';
    return chip;
  }

  function escapeHTML(str) {
    var div = document.createElement("div");
    div.textContent = str;
    return div.innerHTML;
  }

  function escapeAttr(str) {
    return str.replace(/"/g, "&quot;").replace(/'/g, "&#39;");
  }

  // --- State helpers ---
  function getSelectedTags(container) {
    var inputs = container.querySelectorAll('[data-tagcombobox-hidden-inputs] input[type="hidden"]');
    var tags = [];
    for (var i = 0; i < inputs.length; i++) {
      tags.push(inputs[i].value.toLowerCase());
    }
    return tags;
  }

  function isTagSelected(container, name) {
    return getSelectedTags(container).indexOf(name.toLowerCase()) !== -1;
  }

  function addTag(container, name) {
    if (isTagSelected(container, name)) return;

    var disabled = container.querySelector("[data-tagcombobox-text-input]")?.hasAttribute("disabled");
    var chipsContainer = container.querySelector("[data-tagcombobox-chips]");
    var hiddenInputs = container.querySelector("[data-tagcombobox-hidden-inputs]");
    var fieldName = container.getAttribute("data-tagcombobox-name");
    var formAttr = container.getAttribute("data-tagcombobox-form");

    // Add chip
    chipsContainer.appendChild(createChip(name, disabled));

    // Add hidden input
    var input = document.createElement("input");
    input.type = "hidden";
    input.name = fieldName;
    input.value = name;
    if (formAttr) input.setAttribute("form", formAttr);
    hiddenInputs.appendChild(input);

    // Update dropdown checkmark
    updateDropdownState(container);
  }

  function removeTag(container, name) {
    var chip = container.querySelector('[data-tagcombobox-chip="' + CSS.escape(name) + '"]');
    if (chip) chip.remove();

    var hiddenInputs = container.querySelectorAll('[data-tagcombobox-hidden-inputs] input[type="hidden"]');
    for (var i = 0; i < hiddenInputs.length; i++) {
      if (hiddenInputs[i].value.toLowerCase() === name.toLowerCase()) {
        hiddenInputs[i].remove();
        break;
      }
    }

    updateDropdownState(container);
  }

  function updateDropdownState(container) {
    var items = container.querySelectorAll("[data-tagcombobox-item]");
    for (var i = 0; i < items.length; i++) {
      var itemName = items[i].getAttribute("data-tagcombobox-item");
      var check = items[i].querySelector("[data-tagcombobox-check]");
      var selected = isTagSelected(container, itemName);
      if (check) {
        check.classList.toggle("opacity-100", selected);
        check.classList.toggle("opacity-0", !selected);
      }
      items[i].classList.toggle("font-medium", selected);
    }
  }

  // --- Dropdown ---
  function showDropdown(container) {
    var dropdown = container.querySelector("[data-tagcombobox-dropdown]");
    dropdown.classList.remove("hidden");
    filterDropdown(container);
  }

  function hideDropdown(container) {
    var dropdown = container.querySelector("[data-tagcombobox-dropdown]");
    dropdown.classList.add("hidden");
    clearHighlight(container);
  }

  function isDropdownVisible(container) {
    var dropdown = container.querySelector("[data-tagcombobox-dropdown]");
    return !dropdown.classList.contains("hidden");
  }

  function filterDropdown(container) {
    var textInput = container.querySelector("[data-tagcombobox-text-input]");
    var query = (textInput.value || "").toLowerCase().trim();
    var items = container.querySelectorAll("[data-tagcombobox-item]");
    var createOption = container.querySelector("[data-tagcombobox-create]");
    var createLabel = container.querySelector("[data-tagcombobox-create-label]");
    var hasExactMatch = false;
    var visibleCount = 0;

    for (var i = 0; i < items.length; i++) {
      var itemName = items[i].getAttribute("data-tagcombobox-item");
      var matches = query === "" || itemName.toLowerCase().indexOf(query) !== -1;
      items[i].style.display = matches ? "" : "none";
      if (matches) visibleCount++;
      if (itemName.toLowerCase() === query) hasExactMatch = true;
    }

    // Show "Create" option if typed text doesn't exactly match an existing tag
    if (query && !hasExactMatch) {
      createLabel.textContent = query;
      createOption.style.display = "flex";
      createOption.classList.remove("hidden");
      visibleCount++;
    } else {
      createOption.style.display = "none";
      createOption.classList.add("hidden");
    }

    // Show dropdown if items to show
    var dropdown = container.querySelector("[data-tagcombobox-dropdown]");
    if (visibleCount > 0) {
      dropdown.classList.remove("hidden");
    }
  }

  // --- Keyboard navigation ---
  function getVisibleItems(container) {
    var all = container.querySelectorAll("[data-tagcombobox-item], [data-tagcombobox-create]");
    var visible = [];
    for (var i = 0; i < all.length; i++) {
      if (all[i].style.display !== "none" && !all[i].classList.contains("hidden")) {
        visible.push(all[i]);
      }
    }
    return visible;
  }

  function getHighlightedIndex(container) {
    var items = getVisibleItems(container);
    for (var i = 0; i < items.length; i++) {
      if (items[i].hasAttribute("data-tagcombobox-highlighted")) return i;
    }
    return -1;
  }

  function clearHighlight(container) {
    var highlighted = container.querySelectorAll("[data-tagcombobox-highlighted]");
    for (var i = 0; i < highlighted.length; i++) {
      highlighted[i].removeAttribute("data-tagcombobox-highlighted");
      highlighted[i].classList.remove("bg-accent", "text-accent-foreground");
    }
  }

  function highlightItem(container, index) {
    clearHighlight(container);
    var items = getVisibleItems(container);
    if (index >= 0 && index < items.length) {
      items[index].setAttribute("data-tagcombobox-highlighted", "");
      items[index].classList.add("bg-accent", "text-accent-foreground");
      items[index].scrollIntoView({ block: "nearest" });
    }
  }

  function selectHighlighted(container) {
    var items = getVisibleItems(container);
    var index = getHighlightedIndex(container);
    if (index === -1 && items.length > 0) index = 0;
    if (index === -1) return;

    var item = items[index];
    if (item.hasAttribute("data-tagcombobox-create")) {
      // Create new tag
      var label = item.querySelector("[data-tagcombobox-create-label]").textContent;
      addTag(container, label);
    } else {
      var name = item.getAttribute("data-tagcombobox-item");
      if (isTagSelected(container, name)) {
        removeTag(container, name);
      } else {
        addTag(container, name);
      }
    }

    var textInput = container.querySelector("[data-tagcombobox-text-input]");
    textInput.value = "";
    filterDropdown(container);
  }

  // --- Event listeners ---

  // Click on input area -> focus text input
  document.addEventListener("click", function (e) {
    // Remove button
    var removeBtn = e.target.closest("[data-tagcombobox-remove]");
    if (removeBtn && !removeBtn.disabled) {
      e.preventDefault();
      e.stopPropagation();
      var container = removeBtn.closest("[data-tagcombobox]");
      var name = removeBtn.getAttribute("data-tagcombobox-remove");
      removeTag(container, name);
      return;
    }

    // Dropdown item click
    var item = e.target.closest("[data-tagcombobox-item]");
    if (item) {
      e.preventDefault();
      var container = item.closest("[data-tagcombobox]");
      var name = item.getAttribute("data-tagcombobox-item");
      if (isTagSelected(container, name)) {
        removeTag(container, name);
      } else {
        addTag(container, name);
      }
      var textInput = container.querySelector("[data-tagcombobox-text-input]");
      textInput.value = "";
      filterDropdown(container);
      textInput.focus();
      return;
    }

    // Create option click
    var createOpt = e.target.closest("[data-tagcombobox-create]");
    if (createOpt) {
      e.preventDefault();
      var container = createOpt.closest("[data-tagcombobox]");
      var label = createOpt.querySelector("[data-tagcombobox-create-label]").textContent;
      addTag(container, label);
      var textInput = container.querySelector("[data-tagcombobox-text-input]");
      textInput.value = "";
      filterDropdown(container);
      textInput.focus();
      return;
    }

    // Click on input area -> focus
    var inputArea = e.target.closest("[data-tagcombobox-input-area]");
    if (inputArea) {
      var container = inputArea.closest("[data-tagcombobox]");
      var textInput = container.querySelector("[data-tagcombobox-text-input]");
      if (textInput && !textInput.disabled) {
        textInput.focus();
      }
      return;
    }

    // Click outside -> close all dropdowns
    var allContainers = document.querySelectorAll("[data-tagcombobox]");
    for (var i = 0; i < allContainers.length; i++) {
      if (!allContainers[i].contains(e.target)) {
        hideDropdown(allContainers[i]);
      }
    }
  });

  // Focus -> show dropdown
  document.addEventListener("focusin", function (e) {
    var textInput = e.target.closest("[data-tagcombobox-text-input]");
    if (!textInput) return;
    var container = textInput.closest("[data-tagcombobox]");
    if (container) {
      updateDropdownState(container);
      showDropdown(container);
    }
  });

  // Input -> filter
  document.addEventListener("input", function (e) {
    var textInput = e.target.closest("[data-tagcombobox-text-input]");
    if (!textInput) return;
    var container = textInput.closest("[data-tagcombobox]");
    if (container) {
      filterDropdown(container);
      clearHighlight(container);
    }
  });

  // Keyboard navigation
  document.addEventListener("keydown", function (e) {
    var textInput = e.target.closest("[data-tagcombobox-text-input]");
    if (!textInput) return;
    var container = textInput.closest("[data-tagcombobox]");
    if (!container) return;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      if (!isDropdownVisible(container)) {
        showDropdown(container);
        return;
      }
      var items = getVisibleItems(container);
      var idx = getHighlightedIndex(container);
      highlightItem(container, Math.min(idx + 1, items.length - 1));
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      var idx = getHighlightedIndex(container);
      if (idx > 0) {
        highlightItem(container, idx - 1);
      }
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (isDropdownVisible(container)) {
        var idx = getHighlightedIndex(container);
        if (idx >= 0) {
          selectHighlighted(container);
        } else {
          // If text typed but nothing highlighted, try create or select first match
          var query = textInput.value.trim();
          if (query) {
            var createOpt = container.querySelector("[data-tagcombobox-create]");
            if (createOpt && createOpt.style.display !== "none" && !createOpt.classList.contains("hidden")) {
              addTag(container, query);
              textInput.value = "";
              filterDropdown(container);
            } else {
              // Select first visible item
              var items = getVisibleItems(container);
              if (items.length > 0) {
                highlightItem(container, 0);
                selectHighlighted(container);
              }
            }
          }
        }
      }
    } else if (e.key === "Escape") {
      hideDropdown(container);
    } else if (e.key === "Backspace" && textInput.value === "") {
      // Remove last chip
      var chips = container.querySelectorAll("[data-tagcombobox-chip]");
      if (chips.length > 0) {
        var lastChip = chips[chips.length - 1];
        var name = lastChip.getAttribute("data-tagcombobox-chip");
        removeTag(container, name);
      }
    }
  });

  // Form reset
  document.addEventListener("reset", function (e) {
    if (!e.target.matches("form")) return;
    e.target.querySelectorAll("[data-tagcombobox]").forEach(function (container) {
      container.querySelectorAll("[data-tagcombobox-chip]").forEach(function (chip) {
        chip.remove();
      });
      container.querySelectorAll('[data-tagcombobox-hidden-inputs] input[type="hidden"]').forEach(function (inp) {
        inp.remove();
      });
      var textInput = container.querySelector("[data-tagcombobox-text-input]");
      if (textInput) textInput.value = "";
      updateDropdownState(container);
      hideDropdown(container);
    });
  });
})();
