(function () {
  var userMenuButton = document.getElementById('avatar-btn');
  var userDropdown = document.getElementById('user-dropdown');
  if (userMenuButton && userDropdown) {
    userMenuButton.addEventListener('click', function (event) {
      event.stopPropagation();
      userDropdown.classList.toggle('hidden');
    });
    document.addEventListener('click', function () {
      userDropdown.classList.add('hidden');
    });
  }

  var dropdown = document.getElementById('action-dropdown');
  if (dropdown) {
    var currentName = null;
    var currentUrl = null;

    document.querySelectorAll('.action-btn').forEach(function (button) {
      button.addEventListener('click', function (event) {
        event.stopPropagation();
        currentName = button.dataset.name;
        currentUrl = button.dataset.url;

        var rect = button.getBoundingClientRect();
        dropdown.classList.remove('hidden');

        var margin = 8;
        var menuWidth = dropdown.offsetWidth;
        var menuHeight = dropdown.offsetHeight;
        var left = Math.min(rect.left, window.innerWidth - menuWidth - margin);
        var top = rect.bottom + 4;
        if (top + menuHeight > window.innerHeight - margin) {
          top = rect.top - menuHeight - 4;
        }
        dropdown.style.left = Math.max(margin, left) + 'px';
        dropdown.style.top = Math.max(margin, top) + 'px';
      });
    });

    var renameButton = document.getElementById('action-rename');
    if (renameButton) {
      renameButton.addEventListener('click', function () {
        dropdown.classList.add('hidden');
        if (!currentUrl) return;
        var newName = prompt('Rename "' + currentName + '" to:', currentName);
        if (!newName || newName === currentName) return;
        var base = window.location.pathname;
        if (!base.endsWith('/')) base += '/';
        var destination = window.location.origin + base + encodeURIComponent(newName);
        fetch(currentUrl, {
          method: 'MOVE',
          credentials: 'same-origin',
          headers: { Destination: destination, Overwrite: 'F' }
        }).then(function (response) {
          if (response.ok) location.reload();
          else alert('Rename failed: ' + response.status + ' ' + response.statusText);
        }).catch(function (error) {
          alert('Rename failed: ' + error);
        });
      });
    }

    var deleteButton = document.getElementById('action-delete');
    if (deleteButton) {
      deleteButton.addEventListener('click', function () {
        dropdown.classList.add('hidden');
        if (!currentUrl || !confirm('Delete "' + currentName + '"?')) return;
        fetch(currentUrl, { method: 'DELETE', credentials: 'same-origin' })
          .then(function (response) {
            if (response.ok) location.reload();
            else alert('Delete failed: ' + response.status + ' ' + response.statusText);
          }).catch(function (error) {
            alert('Delete failed: ' + error);
          });
      });
    }

    document.addEventListener('click', function () {
      dropdown.classList.add('hidden');
    });
  }

  var uploadButton = document.getElementById('upload-btn');
  var uploadInput = document.getElementById('upload-input');
  if (uploadButton && uploadInput) {
    uploadButton.addEventListener('click', function () {
      uploadInput.click();
    });
    uploadInput.addEventListener('change', function () {
      var files = Array.from(uploadInput.files);
      if (!files.length) return;
      var base = window.location.pathname;
      if (!base.endsWith('/')) base += '/';
      uploadButton.disabled = true;
      var done = 0;
      function updateLabel() {
        uploadButton.textContent = 'Uploading ' + done + '/' + files.length + '…';
      }
      updateLabel();
      Promise.all(files.map(function (file) {
        return fetch(base + encodeURIComponent(file.name), {
          method: 'PUT', credentials: 'same-origin', body: file
        }).then(function (response) {
          if (!response.ok) {
            throw new Error('"' + file.name + '" failed: ' + response.status + ' ' + response.statusText);
          }
          done++;
          updateLabel();
        });
      })).then(function () {
        location.reload();
      }).catch(function (error) {
        alert('Upload failed: ' + error);
        uploadButton.disabled = false;
        uploadButton.innerHTML = '<i class="fa-solid fa-upload"></i> Upload';
        uploadInput.value = '';
      });
    });
  }
})();
