package widgets

import (
	"fmt"

	"github.com/gizak/termui"
)

type ScrollableList struct {
	*termui.List

	SourceItems    []string
	CurrentItem    int
	HighlightColor string

	scroll           int
	highlightedItems []string
}

func NewScrollableList() *ScrollableList {
	return &ScrollableList{
		List:           termui.NewList(),
		CurrentItem:    -1,
		HighlightColor: "blue",
	}
}

func (sl *ScrollableList) SetItems(items []string) {
	sl.SourceItems = make([]string, len(items))
	copy(sl.SourceItems, items)
	sl.highlightedItems = make([]string, len(items))
	copy(sl.highlightedItems, items)
}

func (sl *ScrollableList) Select(item int) {
	if item == sl.CurrentItem {
		return
	}

	count := len(sl.SourceItems)

	if sl.CurrentItem >= 0 && sl.CurrentItem < len(sl.SourceItems) {
		sl.highlightedItems[sl.CurrentItem] = sl.SourceItems[sl.CurrentItem]
	}
	if item >= 0 && item < count {
		sl.highlightedItems[item] =
			fmt.Sprintf("[%-*s](bg-%s)",
				sl.InnerWidth()-1, clearStyles(sl.SourceItems[item]), sl.HighlightColor)

		// Scroll up
		if item <= sl.scroll && item != 0 {
			sl.scroll = item - 1
		}
		// Scroll down
		if item >= sl.scroll+sl.InnerHeight()-2 && item != count-1 {
			sl.scroll = item + 2 - sl.InnerHeight()
		}

		lastItem := sl.scroll + sl.InnerHeight()
		if lastItem > len(sl.SourceItems) {
			lastItem = len(sl.SourceItems)
		}
		sl.Items = sl.highlightedItems[sl.scroll:lastItem]
	}

	sl.CurrentItem = item
}

func (sl *ScrollableList) SelectNext() {
	sl.Select(bound(sl.CurrentItem+1, sl.SourceItems))
}

func (sl *ScrollableList) SelectPrevious() {
	sl.Select(bound(sl.CurrentItem-1, sl.SourceItems))
}
