local Blitbuffer = require("ffi/blitbuffer")
local Button = require("ui/widget/button")
local CenterContainer = require("ui/widget/container/centercontainer")
local Device = require("device")
local Font = require("ui/font")
local FrameContainer = require("ui/widget/container/framecontainer")
local HorizontalGroup = require("ui/widget/horizontalgroup")
local HorizontalSpan = require("ui/widget/horizontalspan")
local InfoMessage = require("ui/widget/infomessage")
local InputContainer = require("ui/widget/container/inputcontainer")
local QRWidget = require("ui/widget/qrwidget")
local Size = require("ui/size")
local TextBoxWidget = require("ui/widget/textboxwidget")
local TextWidget = require("ui/widget/textwidget")
local UIManager = require("ui/uimanager")
local VerticalGroup = require("ui/widget/verticalgroup")
local VerticalSpan = require("ui/widget/verticalspan")
local WidgetContainer = require("ui/widget/container/widgetcontainer")
local _ = require("gettext")
local Screen = Device.screen

local BIN = "/mnt/us/wallpapers/wallpapers"
local TOKEN_FILE = "/tmp/wallpapers.token"
local LOG_FILE = "/tmp/wallpapers.log"
local PORT = 6969

local function readFile(path)
	local f = io.open(path, "r")
	if not f then
		return nil
	end
	local s = f:read("*a")
	f:close()
	return s and s:gsub("%s+$", "") or nil
end

-- Liveness probe. Checks whether the port answers /ping.
local function isRunning()
	return os.execute("wget -q -T 1 -O /dev/null http://127.0.0.1:" .. PORT .. "/ping 2>/dev/null") == 0
end

-- Ask the server who it is. Returns the public URL or nil.
local function fetchURL()
	local tok = readFile(TOKEN_FILE)
	if not tok or tok == "" then
		return nil
	end
	local h = io.popen("wget -q -T 1 -O - http://127.0.0.1:" .. PORT .. "/" .. tok .. "/url 2>/dev/null")
	if not h then
		return nil
	end
	local url = h:read("*a")
	h:close()
	if not url then
		return nil
	end
	url = url:gsub("%s+$", "")
	if url == "" then
		return nil
	end
	return url
end

local function killRunning()
	os.execute("wget -q -O /dev/null --post-data='' http://127.0.0.1:" .. PORT .. "/kill 2>/dev/null")
end

-- ---------------------------------------------------------------------------
-- Panel
-- ---------------------------------------------------------------------------

local Panel = InputContainer:extend({
	modal = true,
	url = nil,
	plugin = nil,
})

function Panel:init()
	local screen_w, screen_h = Screen:getWidth(), Screen:getHeight()
	local pad = Size.padding.fullscreen
	local content_w = screen_w - 4 * pad
	local span_sm = Screen:scaleBySize(10)
	local span_md = Screen:scaleBySize(20)
	local span_lg = Screen:scaleBySize(36)

	local children = {
		TextWidget:new({
			text = self.url and _("Wallpaper server — running") or _("Wallpaper server"),
			face = Font:getFace("tfont", 28),
		}),
		VerticalSpan:new({ width = span_lg }),
	}

	if self.url then
		local qr_size = math.floor(math.min(screen_w, screen_h) * 2 / 3)
		table.insert(
			children,
			QRWidget:new({
				text = self.url,
				width = qr_size,
				height = qr_size,
			})
		)
		table.insert(children, VerticalSpan:new({ width = span_md }))
		table.insert(
			children,
			TextBoxWidget:new({
				text = self.url,
				face = Font:getFace("infofont", 20),
				width = content_w,
				alignment = "center",
			})
		)
		table.insert(children, VerticalSpan:new({ width = span_sm }))
		table.insert(
			children,
			TextBoxWidget:new({
				text = _("Scan the QR or open this URL on a phone on the same wifi."),
				face = Font:getFace("infofont", 16),
				width = content_w,
				alignment = "center",
			})
		)
		table.insert(children, VerticalSpan:new({ width = span_lg }))
		local panel = self
		table.insert(
			children,
			HorizontalGroup:new({
				Button:new({
					text = _("Stop server"),
					callback = function()
						panel.plugin:stop(panel)
					end,
				}),
				HorizontalSpan:new({ width = span_md }),
				Button:new({
					text = _("Close"),
					callback = function()
						UIManager:close(panel)
					end,
				}),
			})
		)
	else
		table.insert(
			children,
			TextBoxWidget:new({
				text = _("Server not running."),
				face = Font:getFace("infofont", 20),
				width = content_w,
				alignment = "center",
			})
		)
		table.insert(children, VerticalSpan:new({ width = span_lg }))
		local panel = self
		table.insert(
			children,
			HorizontalGroup:new({
				Button:new({
					text = _("Start server"),
					callback = function()
						panel.plugin:start(panel)
					end,
				}),
				HorizontalSpan:new({ width = span_md }),
				Button:new({
					text = _("Cancel"),
					callback = function()
						UIManager:close(panel)
					end,
				}),
			})
		)
	end

	local content = VerticalGroup:new({ align = "center" })
	for _, c in ipairs(children) do
		table.insert(content, c)
	end

	local frame = FrameContainer:new({
		background = Blitbuffer.COLOR_WHITE,
		padding = pad,
		content,
	})

	self[1] = CenterContainer:new({
		dimen = Screen:getSize(),
		frame,
	})
end

function Panel:onShow()
	UIManager:setDirty(self, function()
		return "ui", self[1][1].dimen
	end)
	return true
end

function Panel:onCloseWidget()
	UIManager:setDirty(nil, function()
		return "ui", self[1][1].dimen
	end)
end

-- ---------------------------------------------------------------------------
-- Wallpapers plugin
-- ---------------------------------------------------------------------------

local Wallpapers = WidgetContainer:extend({
	name = "wallpapers",
})

function Wallpapers:init()
	self.ui.menu:registerToMainMenu(self)
end

function Wallpapers:addToMainMenu(menu_items)
	menu_items.wallpapers = {
		text_func = function()
			return isRunning() and _("Wallpapers (running)") or _("Wallpapers")
		end,
		callback = function()
			self:open()
		end,
	}
end

function Wallpapers:open()
	if isRunning() then
		UIManager:show(Panel:new({ url = fetchURL(), plugin = self }))
	else
		self:start(nil)
	end
end

function Wallpapers:start(panel)
	if panel then
		UIManager:close(panel)
	end
	if isRunning() then
		UIManager:show(Panel:new({ url = fetchURL(), plugin = self }))
		return
	end
	self.loading = InfoMessage:new({ text = _("Starting wallpaper server…") })
	UIManager:show(self.loading)

	os.execute(BIN .. " >" .. LOG_FILE .. " 2>&1 &")
	self.tries = 0
	UIManager:scheduleIn(0.2, function()
		self:pollForReady()
	end)
end

function Wallpapers:pollForReady()
	if isRunning() then
		local url = fetchURL()
		if url then
			if self.loading then
				UIManager:close(self.loading)
				self.loading = nil
			end
			UIManager:show(Panel:new({ url = url, plugin = self }))
			return
		end
	end
	self.tries = self.tries + 1
	if self.tries > 25 then
		if self.loading then
			UIManager:close(self.loading)
			self.loading = nil
		end
		UIManager:show(InfoMessage:new({ text = _("Server failed to start. Check /tmp/wallpapers.log") }))
		return
	end
	UIManager:scheduleIn(0.2, function()
		self:pollForReady()
	end)
end

function Wallpapers:stop(panel)
	if panel then
		UIManager:close(panel)
	end
	killRunning()
	UIManager:show(Panel:new({ url = nil, plugin = self }))
end

return Wallpapers
