# Software Ergonomics: Why Developer Experience is the Foundation of Great User Experience

As we develop software, we want to deliver products with a good user experience. In reality, most softwares greatly lack in user experience as we focus too much on performance and the rest becomes an afterthought. Software developpers need to focus on bringing back a great UX and to achieve that we need to put back the most important users in the center: **the developers themselves**.

This oversight creates a cascade of problems: overwhelm by complex systems, endless urgent support work that keeps on increasing, tools that fight against us, hypersiloization of team members. In short, developers lose joy in their craft and morale plummets, leading to employee dissatisfaction, resignations, and increased tribal knowledge. The remaining team members face mounting pressure while managers scramble through interview cycles, creating a downward spiral that ultimately results into productivity and delivery slowdown and lower product quality. New hires come in and the integration is challenging, difficult and draining. Lost in all the ecosystems, colleagues are overwhelmed and can't help out. The new hires, often young developers, wonder if it's their fault increasing their sense of imposter syndromes.

How can we ensure morale and joy stay up? It all begins with the workbench.

> Years ago, in one aerospace company I worked for, we hired a new manufacturing floor head engineer to help boost productivity. He was quick to point out that in many work stations, the workers had their backs bent constantly. Within a week, all the stations were corrected. His first few months were all about the ergonomics of the workstations for the employees; the results were transformative. Staff morale increased and the trust they had for their manager allowed them to give their opinion, resulting in a ton of changes. That year, they reached production goals that the previous manufacturing head could not.

How can we translate this to Software? Here comes Software Ergonomics or simply DX for Developer Experience.

## The Golden Rule: One command, Start to work

The first thing to focus on, like the head of the manufacturing floor, is the workstation of the developers. It needs to be optimized by building the required tools and environment to allow good and clean workflow. Whether for development or operations, everything should follow one simple principle: one command, start to work.

The goal is for developers to have everything they need to work at their fingertips with minimal setup. This includes tooling, supporting infrastructure, services, configuration, documentation, seeding, etc.

### Eliminate Tooling Friction

The first barrier to productivity is often the tools themselves. Projects frequently require numerous complicated tools that are difficult to install and use, creating an unnecessary barrier to entry. Remember, developers have varying levels of expertise: some excel at OS level while others don't, some are juniors while others are seniors, and some view computer science as life while others see it just as a job.

A solution is to provide automatic tool installation through:

- Install scripts attached to each project
- Dev containers that encapsulate the entire development environment
- Debian (or other) packages containing all necessary binaries for development and operations

This approach removes barriers and lets developers focus on learning the tools and integrating with their team workflow, regardless of their experience level. Being able to install all the required tools with just one command, speeds up the setup process dramatically improving productivity.

### Conquer Setup Complexity

Modern software systems are dependency nightmares: databases, message buses, cloud services, and swarms of interconnected services. This complexity creates productivity bottlenecks during startup and context-switching, making work feel like climbing a mountain before you can even begin.

It leads to employees avoiding work on certain projects or always pushing it off. Even harder when a project is left untouched for weeks or months and you have to "dust it off". Never mind new hires who barely know the company ecosystem. With time, it leads to increased siloization of individuals as some projects are too complex. Documentation or setup steps tend to be obselete and scattered all over the place if they even exist.

Our approach is to include everything needed in the repository with local-first design:

- Docker Compose infrastructure: Launch supporting services with one command. Each container must be configured properly on localhost and integrated with the local code and other services.
- Local code: Each service **default** configuration should be set up on localhost and work with the supporting infrastructure seamlessly.
- Hot reloading: File watchers that restart services as code changes
- Multiple workflow options: Tie everything together so it can be launched in one command: code, infrastructure, etc. Use Makefile/Justfile commands, TMUX scripts for terminal layouts, VS Code tasks, etc. It is important that each developer enjoys their preferred workflow.

The benefits cascade throughout the development lifecycle. New or returning developers onboard effortlessly instead of spending days fighting setup issues. Simple, smooth workflows keep teams engaged and motivated over time. Well-organized configuration becomes living documentation that teaches system architecture through hands-on experience. Most importantly, clean local setup patterns translate directly to production environments, reducing deployment issues and surprises.

The setup requires, as with anything, ongoing iteration as systems evolve, but the investment pays as well as provides many indirect benefits: team confidence, system understanding, and operational readiness—creating lasting advantages. Make this approach your own by using the tools and setup you like, but **one command, start to work**.

### Lay the Operational Foundation Early

A critical mistake is treating operational concerns as an afterthought. Creating a proper Operation Experience (OX) for the operation teams whether is IT, DevOps or the developers themselves is as important then DX or UX. Operation teams must manage hundreds of different software systems, each with its own configuration approach, documentation quality, bugs, and community support. This easily leads to operational chaos and nightmares that drains productivity and morale. Therefore, it is essential that systems deliver excellent operations experience: Metrics & Dashboards, Structural Loggings, Health Endpoints, AutoReconnect between services, Configuration, Pipelines, etc.

All these elements are simple but impactful. They're can be hard to retrofit due to required code changes, so add them early and evolve them with your system. It will help catch bugs and integration issues when they are the cheapest and easiest to fix. It will make the Operational Experience feel like a breeze increasing joy and ease of usage for both developers and operators.

#### **Dashboards open your system**

Provide system insights through solutions like Prometheus, Datadog or else. Implement them early, even without any interesting metrics as the setup will be ready as features are built. Build dashboards alongside the metrics and logs to make the information visible for developers, maintainers, and operators. Give insight into your system.

#### **Documentation should live in your application**

Documentation should live with your application, not on distant documentation platforms where it gets lost and forgotten. If the documentation live with the code, it can be improved and updated as the code changes. Make it part of merge or pull request. Moreover, many markdown-to-HTML solutions enable great documentation websites with markdown flexibility.

#### **Pipelines keep the system in check**

CI/CD is primordial in modern software but often left until the end or abandoned entirely.

> I once had to integrate Python software from a company we bought into our ecosystem with a week deadline for an $17 million contract presentation. The published containers failed on startup, the codebase wouldn't start locally, and the Dockerfile was broken. After a week of fixes, we discovered their pipeline had been broken for 9 months, required three repositories to run locally, and it needed my fixes to their Dockerfile and code base to run properly.
>
> We managed to make it work with their team after a full week of hard work, but their software was sending the data in the wrong format so we had to cut them out of the presentation anyway.

CI pipelines should exist in early phases, even with just builds and dummy unit tests. It helps validate the reproductibility of the setup and is a source of documentation in itself. It will help catch bugs and integration issues immediately when they are the cheapest and easiest to fix. As the project progresses, you keep on adding to it: containers, testing, deployment, etc. The pipeline helps control and keep in check the evolution of the codebase.

#### **Auto-Reconnect reduce operational pain**

Your system is one of many that operations teams manage. It needs resilience against network failures, outages, and other issues. Auto-reconnect prevents cascading failures but needs early implementation as it can change code structure.

> I once built a Kubernetes platform for a "ready" system of 5 sequential services and we had around 30 deployments. Each service was unstable and would crash often. When one crashed, others followed or became unrecoverable, requiring manual restart of each in proper order. Quickly, it became a huge pain to operate as well as greatly reduce trust with our user base. The solution was too add auto-reconnect: 4 services was simple, but the fifth took a month due to poor structure costing a lot of my time and the developers time.

## From DX and OX to UX, Building Systems People Love to Use

In construction, architecture focuses on aesthetic design and how people live in the different spaces, while civil engineering handles technical feasibility and safety. In software, we teach architecture like civil engineering: design patterns, services, performance, etc. There is no focus on how developers, maintainers, operators and users live in that software space. The result is often an over-architected systems with poor usability and ergonomics for all user types.

> In a previous job, a team managed data across hundreds of databases and S3 storage for engineering professionals company-wide. The complexity made it so difficult to manage and use, leading to the infamous company quote: "Where is my data?". A UI was built to help users manage the data they needed, but eventually, the UI was removed from the hands of the users because it was too difficult to operate.
>
> Built by a solo developer, most team members avoided developing or operating the platform. It was complicated, difficult to use, and joyless, leading to hyper-siloization and operational issues; including pulling the developer from vacation when problems arose. At this level, it was not the developer's fault, but a leadership failure.

From this point, we have the basics covered to develop the system. It is easy to get into the workflow and we can be productive rapidly. As we develop the system, what do we do the most? Well, we use it! We trigger different things to test functionality: API endpoints, data seeding, complex button sequences. Try to remember the latest API you worked on, you problably had a complicated API with too many endpoints and too many parameters. In the end, even if you built the API or not, you struggle to use the system as a developer, get lost in the rapidly growing ecosystems and find it difficult to operate, and simply not fun. If we struggle as builders, how will users, operators and even other devs fare? Fast-foward a few months, it goes into production and you, users or operators struggle to use it. If you did build the API, you might be able to use it greatly, because you know it by heart but is it the case for your coworkers and users? This will greatly increase hypersiloization of you and your team members in their respective projects. You need to enjoy the tools you build, you need to bring joy to using your systems.

How do we get around this? We need to provide a user experience by providing flows, not just buttons and switches. As we reduced the barrier of entry to develop in a project, we need to do the same to operate it! To learn how to interact with our system, we need to take incremental steps from low level to high level: automatic testing, CLI and finally a full user interface.

### Automatic Testing, Controlling Volatility

Automatic testing is quite the subject, with many approaches and philosophies, but it has a clear goal and it is not about finding bugs. Software evoles like writing a book: draft after draft, morphing as requirements changes and system grows. Design and architecture must adjust, leading to small or large refactors. Systems that reach crystallization point do so because developers cannot or fear refactoring. Automatic testing ensures system viability despite the necessary changes by giving you confidence to make the changes because the tests will always back you up.

Integrating tests into development might feel slow intially, but productivity enhances afterward. Treat tests as first class citizen! Take the the time to write test utilities and domain specific libraries. If your tests have a lot of boilerplate code, write helpers to it to make it easier. Writing test is as much part of a system as writing core code, you must find joy in it.

While writing tests, think beyond triggering API endpoints; start refining your thinking about the user's workflow, about the CLI and the web/desktop interface. How will the users use the system? Which actions might trigger a few endpoints? Do you need new ones? Do you need to add parameters to existing ones? Take the time to adjust your API with the vision you're building. As you work on the next step, you will be happy that the API is rock-solid and has everything you need to construct that vision.

With automatic testing as part of the development flow, you control a software's volatility, find a lot of initial bugs and glitches and slowly build the vision for the user's workflows.

### CLI, The Stepping Stone

One of the most important thing is to enjoy using the software you are building. If you hate using your own software, you'll build something that everyone else hates too. You create a cycle of not caring as you develop resulting in a poor craft. A CLI isn't just a developer tool, it's your first real user interface. It's where you stop thinking like a backend engineer and start thinking like someone who actually has to get work done. Your CLI is higher-level than raw APIs but lower than a web app. It's where those workflow ideas bouncing around your head while coding finally get tested in reality.

Most CLIs are lazy API wrappers; don't do this. Your CLI should solve actual problems, not just expose endpoints. If your users always repeat a set of operations, combine them into one command. Make the commands interactive if needed, guide the users as they use the CLI. Using a CLI where you have to remember all the commands, the positional arguments and the flags is difficult and unpleasant, make them consistent accross commands. Make the CLI intuitive:

- Need login? Redirect to login automatically
- Need a JSON payload? Generate the template for them
- Need to understand what happened? Show them useful output, not cryptic success codes

What does the user need to accomplish a task and what do they want after they have run the command? Provide those.

Depending on the type of software you are writing, the CLI might become part of a lot of automation. Enable that by providing great integration with piping stdin and stdout through the different commands. For example, if a command needs a JSON payload in an argument, add piping to that argument with something like: `cat payload.json | mir cmd send -n ActivateStartUpSeq`. If your CLI is also the server, you should add commands to help its operation:

- Add a command to print the current configuration
- A command to create the default configuration file at the default location
- Display the loaded configuration (hide secrets) as logs as the software boots up and where it reads the config.

Operations and developers teams manage hundreds of tools. Make yours the one they don't curse at. They will love those little details as they make things so much simpler. IT and DevOps engineers need to be part of the experience.

As you write your CLI, you will keep on building an understanding of the way you want your users to interact with your system which will help you in the design of the next step which is often a web application. Iterate on your API to make sure everything is there for the next step of your vision. A CLI will empower you and your team to operate the system faster, automate tasks, reduce support time and build a great quality product overall. As with everything else, a CLI can be iterated upon! Each new features should come with the new core code, automatic testing and CLI. When developers enjoy their tools, they build better products.

This podcast on CLI design offers deeper reflection on the subject: [GoTime Podcast](https://gotime.fm/337).

## Conclusion

The path to exceptional user experience begins not with the end user, but with the developers sitting at their workstation. Just as with any manufacturing engineer, ergonomics including workbenches, tools, and workflows directly determine the quality of what we build. Software ergonomics isn't just about making developers happy; it's about creating a foundation for sustainable and high-quality software development. When developers struggle with complicated setup processes, fragmented tooling, and systems they themselves find difficult to use, that struggle inevitably propagates to the end product and employee dissatisfaction. The frustrated developer who dreads working on a particular system will not craft the elegant and user-friendly experience. Delight all your users, don't just deliver code.

The incremental approach outlined, from "one command, start to work" to automated testing, CLI development, and finally to full user interfaces, creates a natural progression where each step builds understanding, confidence and a better user experience. This isn't just about process; it's about creating joy in the craft of software development. When developers enjoy using what they're building, when they can easily demonstrate features to colleagues, when onboarding new team members becomes effortless rather than painful, the entire organization benefits. When the DX, OX or UX is not right, there will be disenchantment from every type of user and this creates a downward spiral: support work increases, joy decreases, siloization increases, etc. A beautiful UI does not help or resolve the underlying issues nor does technical perfection:

> In my early days of programming, I worked in a car sharing company with some of the worst software I have ever seen: thousand lines functions, personal information such as credit cards, driver's license numbers written plainly in the database, 800 tables in SQL with 0 foreign keys, and so much more and the UI wasn't pretty either. Nonetheless, operators and users loved the system. Why? Because the developers were extremely close to the user base. You don't need to be an expert developer, you just need to be attentive to your user base. Each developer and operator had each other's back; teamwork makes the dream work. In retrospect, that carsharing company made among the best software system I have seen (after finally encrypting credit cards and personal information).

What matters most is the connection between builders and users. By combining a user-focused mindset with proper developer ergonomics, we can achieve both technical excellence and user satisfaction.

Investing time in developer tooling, ergonomics, and experience isn't a distraction from "real work", it's what makes the real work possible, sustainable, and ultimately successful. Every minute spent improving the developer experience pays dividends in faster delivery, fewer bugs, better user experiences, and higher team morale. The most important users of your software system are the developers. When you take care of them, they'll take care of everyone else. Start with your workbench, build incrementally, and watch as improved developer experience naturally evolves into exceptional user experience. Your future self, your teammates, and your users will thank you for it.

Here's a little story I often relate: In Stephen Covey's influential book "The 7 Habits of Highly Effective People," the seventh habit is from the "Sharpen the Axe" story.

> Two lumberjacks compete to cut the most trees in one day. The first worked non-stop without breaks, while the second worked for an hour then rested for ten minutes every hour. At the day's end, the second lumberjack had cut far more trees. When the first asked how this was possible despite all the breaks, the second replied, "I wasn't just taking break, I was sharpening my axe."

The lesson for software engineers: taking breaks from big features to rest your mind, sharpen your skills and improve tooling isn't slowing you down, it's what ultimately makes you faster, more effective and gives you higher moral. Prioritise the development experience. Proper workbench and setup is about the culture it creates.

> Artisans of an earlier age where proud to sign their work. You should be, too.
> 											- The Pragmatic Programmer, Page 485 -
