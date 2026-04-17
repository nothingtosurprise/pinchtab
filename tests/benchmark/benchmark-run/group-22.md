# Group 22: Mouse Drag & Drop

### 22.1 Drag a piece into Zone A
Navigate to `http://fixtures/drag.html`. The page contains a draggable square (`#piece`) and three target zones (`#zone-a`, `#zone-b`, `#zone-c`). Drag the piece so its center ends up over **Zone A**.

**Verify**: The page shows `LAST_DROP=DROP_ZONE_A_OK`.

### 22.2 Drag to Zone B, then Zone C
Without reloading the page, drag the piece next into **Zone B**, and then into **Zone C**. The page records an ordered drop sequence.

**Verify**: The page shows `DROP_SEQUENCE=DROP_ZONE_A_OK,DROP_ZONE_B_OK,DROP_ZONE_C_OK` (all three drops in order).

---

