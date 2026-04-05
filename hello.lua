tmixer = {}
tmixer.projects = merge(
    {
        projects = {
            directory = "~/Projects",
            windows = {
                {
                    name = "nvim",
                    startup = function()
                        print("Hello window!")
                        return {"nvim"}
                    end,
                    panes = {
                        name = "nvim",
                        startup = function()
                            return {"nvim"}
                        end,
                        split = "horizonal"
                    }
                }
            },
            switch = function()
                print("Switch")
            end
        },
        bin = {
            directory = "~/bin"
        },
        brain = {
            directory = "~/Documents/brainge"
        },
    },
    loadSubDirectories(
        "~/Projects",
        "project--",
        {
            switch = function()
                print("hello sub dir")
                print(os.getenv("OK"))
                return {{"nim", "."}, {""}}
            end
        }
    )
)
